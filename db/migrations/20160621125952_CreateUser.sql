-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

create table  if not exists users (
        id uuid primary key default(uuid_generate_v4()),
        email varchar(255) not null unique,
        full_name varchar(255),
        password bytea,
        secret char(50),
        created_at timestamp without time zone default(CURRENT_TIMESTAMP at time zone 'utc')
        );
-- +goose StatementEnd


-- +goose Down
drop table users;
