#!/bin/bash

# Usage: adduser.sh [db] username

if [ $# -eq 1 ]; then
    db="./lxcpanel.sqlite3"
    username="$1"
else
    db="$1"
    username="$2"
fi

if [ $(sqlite3 $db "SELECT COUNT(*) FROM users WHERE username='$username';") -ne 0 ]; then
    echo "User already exists"
    exit 1
fi

sqlite3 $db "INSERT INTO users (username) VALUES ('$username');"
if [ $? -eq 0 ]; then
    echo "User added successfully"
else
    echo "Failed to add user"
fi
