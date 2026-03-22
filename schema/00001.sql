CREATE TYPE locker_status AS ENUM ('available', 'in_use', 'maintenance');
CREATE TYPE session_status AS ENUM ('pending_payment', 'charging', 'completed', 'expired');

CREATE TABLE locker (
    id         bigserial     PRIMARY KEY,
    name       varchar(50)   NOT NULL,
    status     locker_status NOT NULL DEFAULT 'available',
    created_at timestamptz   NOT NULL DEFAULT now(),
    updated_at timestamptz   NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_locker_name ON locker(name);

CREATE TABLE session (
    id           bigserial      PRIMARY KEY,
    locker_id    bigint         NOT NULL REFERENCES locker(id),
    status       session_status NOT NULL DEFAULT 'pending_payment',
    qr_code_data text           NOT NULL DEFAULT '',
    payment_hash text NOT NULL DEFAULT '',
    amount       bigint         NOT NULL DEFAULT 0,
    started_at   timestamptz,
    expired_at   timestamptz,
    created_at   timestamptz    NOT NULL DEFAULT now(),
    updated_at   timestamptz    NOT NULL DEFAULT now()
);

CREATE INDEX idx_session_locker_id ON session(locker_id);
CREATE INDEX idx_session_status ON session(status);
