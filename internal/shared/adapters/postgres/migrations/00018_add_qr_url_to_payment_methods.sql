-- +goose Up
ALTER TABLE orders.payment_methods ADD COLUMN IF NOT EXISTS qr_url TEXT;

UPDATE orders.payment_methods
SET qr_url = 'https://res.cloudinary.com/dxmvyitlc/image/upload/v1780021364/WhatsApp_Image_2026-05-28_at_9.40.25_PM_hst1le.jpg'
WHERE type = 'PagoMovil';

-- +goose Down
ALTER TABLE orders.payment_methods DROP COLUMN IF EXISTS qr_url;
