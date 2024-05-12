#!/bin/bash

# Usage: create_admin.sh

# Prompt for database path
read -p "Enter database path (default: ./lxcpanel.sqlite3): " db
db="${db:-./lxcpanel.sqlite3}"

# Prompt for admin username
read -p "Enter admin username: " username

# Prompt for admin public key
read -p "Enter admin public key: " pubkey

# Check if admin already exists
if [ $(sqlite3 $db "SELECT COUNT(*) FROM users WHERE username='$username';") -ne 0 ]; then
    echo "User already exists"
    exit 1
fi

# Insert admin into the database
sqlite3 $db "INSERT INTO users (username, admin) VALUES ('$username', TRUE);"
if [ $? -eq 0 ]; then
    echo "User added successfully"
else
    echo "Failed to add user"
    exit 1
fi

fingerprint=$(echo -n $pubkey | shasum -a 256 | cut -d ' ' -f 1)
# Insert public key into the database
sqlite3 $db "INSERT INTO pubkeys (fingerprint, username, pubkey) VALUES ('$fingerprint', '$username', '$pubkey');"
if [ $? -eq 0 ]; then
    echo "Public key added successfully"
else
    echo "Failed to add public key"
    exit 1
fi

