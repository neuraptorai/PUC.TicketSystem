-- File: sql/schema.sql

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), 
    oauth_provider VARCHAR(50),
    oauth_id VARCHAR(255),
    created_at TIMESTAMMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(oauth_provider, oauth_id)
);

CREATE TABLE IF NOT EXISTS reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    event_id VARCHAR(255) NOT NULL,
    ticket_type_id VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    status VARCHAR(50) NOT NULL, -- RESERVED, CONVERTED, EXPIRED
    created_at TIMESTAMMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    reservation_id UUID NOT NULL REFERENCES reservations(id),
    status VARCHAR(50) NOT NULL, -- COMPLETED, FAILED
    total_amount NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMMPTZ NOT NULL DEFAULT NOW()
);