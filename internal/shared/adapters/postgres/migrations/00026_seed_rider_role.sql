-- +goose Up
INSERT INTO users.roles (name)
VALUES ('Rider')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM users.roles WHERE name = 'Rider';
