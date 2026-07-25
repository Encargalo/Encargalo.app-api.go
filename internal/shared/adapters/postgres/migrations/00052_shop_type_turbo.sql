-- +goose Up
ALTER TABLE business.shops DROP CONSTRAINT IF EXISTS shops_type_check;
ALTER TABLE business.shops
    ADD CONSTRAINT shops_type_check
    CHECK ((type)::TEXT = ANY (ARRAY['Restaurant'::TEXT, 'Supermarket'::TEXT, 'Turbo'::TEXT]));

-- +goose Down
ALTER TABLE business.shops DROP CONSTRAINT IF EXISTS shops_type_check;
ALTER TABLE business.shops
    ADD CONSTRAINT shops_type_check
    CHECK ((type)::TEXT = ANY (ARRAY['Restaurant'::TEXT, 'Supermarket'::TEXT]));
