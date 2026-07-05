-- +goose Up
CREATE TABLE products.schedules (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id UUID        REFERENCES products.categories(id) ON DELETE CASCADE,
    item_id     UUID        REFERENCES products.items(id) ON DELETE CASCADE,
    day_of_week SMALLINT    CHECK (day_of_week BETWEEN 0 AND 6),
    start_time  TIME        NOT NULL,
    end_time    TIME        NOT NULL,
    enabled     BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_schedules_target CHECK (
        (category_id IS NOT NULL AND item_id IS NULL) OR
        (category_id IS NULL AND item_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_schedules_category_id ON products.schedules (category_id);
CREATE INDEX IF NOT EXISTS idx_schedules_item_id ON products.schedules (item_id);

-- +goose Down
DROP INDEX IF EXISTS idx_schedules_item_id;
DROP INDEX IF EXISTS idx_schedules_category_id;
DROP TABLE IF EXISTS products.schedules;
