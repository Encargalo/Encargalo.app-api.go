-- +goose Up
ALTER TABLE business.shops
    ADD COLUMN first_opened_at TIMESTAMPTZ,
    ADD COLUMN launch_promo_ends_at TIMESTAMPTZ,
    ADD COLUMN zero_commission_ends_at TIMESTAMPTZ;

-- Backfill de negocios existentes con ambas ventanas ya vencidas, para que
-- no reciban las promos de lanzamiento retroactivamente al desplegar.
UPDATE business.shops
SET first_opened_at = created_at,
    launch_promo_ends_at = created_at,
    zero_commission_ends_at = created_at
WHERE first_opened_at IS NULL;

ALTER TABLE orders.orders
    ADD COLUMN is_launch_promo BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_zero_commission BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN promo_subsidy_amount NUMERIC(10,2) NOT NULL DEFAULT 0;

CREATE TABLE products.item_discounts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id       UUID NOT NULL REFERENCES products.items(id),
    shop_id       UUID NOT NULL REFERENCES business.shops(id),
    discount_type TEXT NOT NULL CHECK (discount_type IN ('percentage','fixed')),
    value         NUMERIC(10,2) NOT NULL,
    starts_at     TIMESTAMPTZ,
    ends_at       TIMESTAMPTZ,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_item_discounts_item_id ON products.item_discounts (item_id);
CREATE INDEX idx_item_discounts_shop_id ON products.item_discounts (shop_id);

-- Un solo descuento activo por producto.
CREATE UNIQUE INDEX ux_item_discounts_active_item
    ON products.item_discounts (item_id) WHERE is_active AND deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS ux_item_discounts_active_item;
DROP INDEX IF EXISTS idx_item_discounts_shop_id;
DROP INDEX IF EXISTS idx_item_discounts_item_id;
DROP TABLE IF EXISTS products.item_discounts;

ALTER TABLE orders.orders
    DROP COLUMN IF EXISTS is_launch_promo,
    DROP COLUMN IF EXISTS is_zero_commission,
    DROP COLUMN IF EXISTS promo_subsidy_amount;

ALTER TABLE business.shops
    DROP COLUMN IF EXISTS first_opened_at,
    DROP COLUMN IF EXISTS launch_promo_ends_at,
    DROP COLUMN IF EXISTS zero_commission_ends_at;
