CREATE TABLE IF NOT EXISTS users (
    username VARCHAR(50) NOT NULL PRIMARY KEY,
    max_instance_count INTEGER NOT NULL DEFAULT 3,
    admin BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS pubkeys (
    fingerprint VARCHAR(64) NOT NULL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    pubkey TEXT NOT NULL,
    FOREIGN KEY (username) REFERENCES users(username)
);
