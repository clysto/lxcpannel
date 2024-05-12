PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    username VARCHAR(50) NOT NULL PRIMARY KEY,
    max_instance_count INTEGER NOT NULL DEFAULT 3,
    admin BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS pubkeys (
    fingerprint VARCHAR(64) NOT NULL,
    username VARCHAR(50) NOT NULL,
    pubkey TEXT NOT NULL,
    PRIMARY KEY (fingerprint, username),
    FOREIGN KEY (username) REFERENCES users(username)
);
