CREATE TABLE IF NOT EXISTS locker (
    id         INTEGER      PRIMARY KEY AUTOINCREMENT,
    name       VARCHAR(50)  NOT NULL,
    status     TEXT         NOT NULL DEFAULT 'available',
    created_at DATETIME     NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME     NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_locker_name ON locker(name);

CREATE TABLE IF NOT EXISTS session (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    locker_id    INTEGER  NOT NULL REFERENCES locker(id),
    status       TEXT     NOT NULL DEFAULT 'pending_payment',
    qr_code_data TEXT     NOT NULL DEFAULT '',
    payment_hash TEXT     NOT NULL DEFAULT '',
    amount       INTEGER  NOT NULL DEFAULT 0,
    started_at   DATETIME,
    expired_at   DATETIME,
    created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at   DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_session_locker_id ON session(locker_id);
CREATE INDEX IF NOT EXISTS idx_session_status ON session(status);
