package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type WithdrawalStatus string

const (
	WithdrawalPending    WithdrawalStatus = "pending"
	WithdrawalProcessing WithdrawalStatus = "processing"
	WithdrawalCompleted  WithdrawalStatus = "completed"
	WithdrawalRejected   WithdrawalStatus = "rejected"
)

const (
	MovementTypeWithdrawalHold       MovementType = "withdrawal_hold"       // pending: saldo retenido
	MovementTypeWithdrawalProcessing MovementType = "withdrawal_processing" // reclamada: monto congelado
	MovementTypeWithdrawalPayout     MovementType = "withdrawal_payout"     // completed: ya transferido
	MovementTypeWithdrawalRestore    MovementType = "withdrawal_restore"    // rejected: saldo devuelto

	ReferenceTypeWithdrawalRequest = "withdrawal_request"
)

type WithdrawalRequest struct {
	bun.BaseModel `bun:"table:finance.withdrawal_requests"`

	ID         uuid.UUID `bun:"id,pk,type:uuid"`
	EntityID   uuid.UUID `bun:"entity_id,type:uuid"`
	EntityType string    `bun:"entity_type"`
	Amount     float64   `bun:"amount"`
	// PayableAmount es el neto congelado al reclamar: lo que el operador vio y
	// transfirió. Nil mientras la solicitud sigue pending.
	PayableAmount   *float64         `bun:"payable_amount,nullzero"`
	Currency        string           `bun:"currency"`
	BcvRate         *float64         `bun:"bcv_rate,nullzero"`
	Status          WithdrawalStatus `bun:"status"`
	PMPhone         *string          `bun:"pm_phone,nullzero"`
	PMBankCode      *string          `bun:"pm_bank_code,nullzero"`
	PMNationalID    *string          `bun:"pm_national_id,nullzero"`
	ProofURL        *string          `bun:"proof_url,nullzero"`
	StaffID         *uuid.UUID       `bun:"staff_id,nullzero,type:uuid"`
	RejectionReason *string          `bun:"rejection_reason,nullzero"`
	CreatedAt       time.Time        `bun:"created_at"`
	ProcessingAt    *time.Time       `bun:"processing_at,nullzero"`
	CompletedAt     *time.Time       `bun:"completed_at,nullzero"`
	ResolvedAt      *time.Time       `bun:"resolved_at,nullzero"`
}

// IsActive reporta si la solicitud sigue ocupando el cupo de solicitud única.
func (w WithdrawalRequest) IsActive() bool {
	return w.Status == WithdrawalPending || w.Status == WithdrawalProcessing
}

// CanTransitionTo valida la máquina de estados. No existe pending -> completed
// directo: toda solicitud pasa por processing, porque es ese paso el que
// congela el monto que se va a transferir (ver design.md, decisión 5).
func (w WithdrawalRequest) CanTransitionTo(next WithdrawalStatus) bool {
	switch w.Status {
	case WithdrawalPending:
		return next == WithdrawalProcessing || next == WithdrawalRejected
	case WithdrawalProcessing:
		return next == WithdrawalCompleted || next == WithdrawalRejected
	default:
		return false
	}
}

// PayableFrom es el neto que la plataforma realmente debe: lo retenido menos
// la deuda que el solicitante acumuló después de pedir el retiro. Un pedido en
// efectivo deja el disponible negativo, y esa deuda se salda contra el propio
// retiro en vez de transferir de más.
func PayableFrom(held, available float64) float64 {
	if available >= 0 {
		return held
	}
	return held + available
}

// WithdrawalCooldown corre desde el pago, no desde la solicitud: el costo que
// la regla acota es la transferencia manual, y esa ocurre al pagar.
const WithdrawalCooldown = 24 * time.Hour

// WithdrawalListRow es la solicitud con los datos del solicitante que el CRM
// necesita, resueltos en el mismo query.
type WithdrawalListRow struct {
	WithdrawalRequest

	Name    string  `bun:"name"`
	SurName *string `bun:"sur_name"`
	Phone   *string `bun:"phone"`
	// AvailableBalance es el saldo disponible actual del solicitante: con él
	// se calcula el pagable de una solicitud que todavía no fue reclamada.
	AvailableBalance float64 `bun:"available_balance"`
}

// Payable es lo que hay que transferir hoy. Una vez reclamada manda el monto
// congelado; antes de eso se recalcula contra el saldo disponible, porque el
// efectivo que el rider cobró mientras tanto reduce la deuda.
func (r WithdrawalListRow) Payable() float64 {
	if r.PayableAmount != nil {
		return *r.PayableAmount
	}
	return PayableFrom(r.Amount, r.AvailableBalance)
}
