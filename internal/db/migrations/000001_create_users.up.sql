CREATE EXTENSION IF NOT EXISTS "pgcrypto"; --use to generate UUIDs with gen_random_uuid() function

CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(20),
    age INTEGER,
    status VARCHAR(8) NOT NULL DEFAULT 'Active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_status_check CHECK (status IN ('Active', 'Inactive')),
    CONSTRAINT users_age_check CHECK (age IS NULL OR age > 0)
);
