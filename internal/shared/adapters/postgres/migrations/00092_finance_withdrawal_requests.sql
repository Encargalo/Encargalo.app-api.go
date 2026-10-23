-- +goose Up
-- El retiro deja de auto-liquidarse. La solicitud vive acá con su estado y el
-- ledger la acompaña: withdrawal_hold (pending) -> withdrawal_processing
-- (reclamada) -> withdrawal_payout (pagada) / withdrawal_restore (rechazada).
-- Genérica por entity_type: los comercios heredan el flujo sin duplicar tabla.
CREATE TABLE IF NOT EXISTS finance.withdrawal_requests (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id        UUID          NOT NULL,
    entity_type      TEXT          NOT NULL,
    amount           NUMERIC(12, 2) NOT NULL,
    -- payable_amount se congela al reclamar: es el neto que el operador vio y
    -- transfirió. NULL mientras la solicitud sigue pending.
    payable_amount   NUMERIC(12, 2),
    currency         TEXT          NOT NULL DEFAULT 'VES',
    bcv_rate         NUMERIC(14, 6),
    status           TEXT          NOT NULL DEFAULT 'pending',
    -- Los pm_* son NULL solo en las filas del backfill, que no tienen de dónde
    -- sacarlos; el app layer exige Pago Móvil antes de crear una solicitud.
    pm_phone         TEXT,
    pm_bank_code     TEXT,
    pm_national_id   TEXT,
    proof_url        TEXT,
    staff_id         UUID REFERENCES users.accounts (id),
    rejection_reason TEXT,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    processing_at    TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    resolved_at      TIMESTAMPTZ
);

-- Una sola solicitud activa por entidad: la regla anti-spam es un constraint,
-- no lógica de aplicación sujeta a carreras.
CREATE UNIQUE INDEX IF NOT EXISTS uq_withdrawal_requests_entity_active
    ON finance.withdrawal_requests (entity_id, entity_type)
    WHERE status IN ('pending', 'processing');

-- Listado del CRM por estado.
CREATE INDEX IF NOT EXISTS idx_withdrawal_requests_status
    ON finance.withdrawal_requests (status, created_at DESC);

-- Último retiro pagado de una entidad (cooldown) e historial por entidad.
CREATE INDEX IF NOT EXISTS idx_withdrawal_requests_entity
    ON finance.withdrawal_requests (entity_id, entity_type, created_at DESC);

-- Una sola reserva de saldo por solicitud: el saldo no puede retenerse ni
-- liberarse dos veces por el mismo retiro. Espejo de la de reembolsos.
CREATE UNIQUE INDEX IF NOT EXISTS uq_finance_tx_withdrawal_reservation
    ON finance.transactions (reference_id)
    WHERE reference_type = 'withdrawal_request'
      AND movement_type IN ('withdrawal_hold', 'withdrawal_processing', 'withdrawal_payout');

-- Backfill: los movimientos `withdrawal` existentes son datos de prueba (la
-- plataforma no opera todavía), así que se registran como solicitudes ya
-- pagadas sin comprobante, para no dejar movimientos sin solicitud que los
-- explique. La solicitud reusa el UUID del movimiento como id, lo que da la
-- correspondencia sin tabla intermedia.
INSERT INTO finance.withdrawal_requests (
    id, entity_id, entity_type, amount, payable_amount, currency, bcv_rate,
    status, created_at, processing_at, completed_at, resolved_at
)
SELECT t.id,
       t.entity_id,
       t.entity_type,
       -t.amount,
       -t.amount,
       COALESCE(NULLIF(t.currency, ''), 'VES'),
       t.bcv_rate,
       'completed',
       t.created_at,
       t.created_at,
       t.created_at,
       t.created_at
FROM finance.transactions t
WHERE t.movement_type = 'withdrawal'
ON CONFLICT (id) DO NOTHING;

UPDATE finance.transactions
SET movement_type  = 'withdrawal_payout',
    reference_id   = id,
    reference_type = 'withdrawal_request'
WHERE movement_type = 'withdrawal';

-- +goose Down
-- Revertir los pagos a su tipo plano. Ojo: un rollback con solicitudes
-- abiertas deja holds sin solicitud que los explique, así que antes de
-- revertir hay que resolver (pagar o rechazar) toda solicitud activa.
UPDATE finance.transactions
SET movement_type  = 'withdrawal',
    reference_id   = NULL,
    reference_type = NULL
WHERE movement_type = 'withdrawal_payout'
  AND reference_type = 'withdrawal_request';

DROP INDEX IF EXISTS finance.uq_finance_tx_withdrawal_reservation;
DROP INDEX IF EXISTS finance.idx_withdrawal_requests_entity;
DROP INDEX IF EXISTS finance.idx_withdrawal_requests_status;
DROP INDEX IF EXISTS finance.uq_withdrawal_requests_entity_active;
DROP TABLE IF EXISTS finance.withdrawal_requests;
