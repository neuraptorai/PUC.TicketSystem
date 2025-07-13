-- File: sql/migrations/000001_create_initial_tables.up.sql

BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    oauth_provider VARCHAR(50),
    oauth_id VARCHAR(255),
    -- Corrigido de TIMESTAMMPTZ para TIMESTAMPTZ
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(oauth_provider, oauth_id)
);

CREATE TABLE IF NOT EXISTS reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    event_id VARCHAR(255) NOT NULL,
    ticket_type_id VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    -- Corrigido de TIMESTAMMPTZ para TIMESTAMPTZ
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Corrigido de TIMESTAMMPTZ para TIMESTAMPTZ
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    reservation_id UUID NOT NULL REFERENCES reservations(id),
    status VARCHAR(50) NOT NULL,
    total_amount NUMERIC(10, 2) NOT NULL,
    -- Corrigido de TIMESTAMMPTZ para TIMESTAMPTZ
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;