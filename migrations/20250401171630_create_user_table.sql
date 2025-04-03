-- +goose Up
-- +goose StatementBegin
CREATE TYPE role_enum AS ENUM ('unspecified', 'user', 'admin');
CREATE TABLE "users" (
    id serial PRIMARY KEY,
    name text NOT NULL,
    email text NOT NULL,
    role role_enum,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "users";
DROP TYPE role_enum;
-- +goose StatementEnd
