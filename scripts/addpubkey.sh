#!/bin/bash

# Usage: addpubkey.sh [db] username pubkey

if [ $# -eq 2 ]; then
    db="./lxcpanel.sqlite3"
    username="$1"
    pubkey="$2"
else
    db="$1"
    username="$2"
    pubkey="$3"
fi

fingerprint=$(echo $pubkey | shasum -a 256 | cut -d ' ' -f 1)

if [ $(sqlite3 $db "SELECT COUNT(*) FROM pubkeys WHERE fingerprint='$fingerprint';") -ne 0 ]; then
    echo "Public key already exists"
    exit 1
fi

sqlite3 $db "INSERT INTO pubkeys (fingerprint, username, pubkey) VALUES ('$fingerprint', '$username', '$pubkey');"
echo "Public key added successfully"
