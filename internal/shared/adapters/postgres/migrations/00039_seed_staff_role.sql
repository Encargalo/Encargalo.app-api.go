-- +goose Up
INSERT INTO users.roles (name)
VALUES ('Staff')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM users.roles WHERE name = 'Staff';
