package common

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed init.sql
var initSQL string

type DBPubKey struct {
	Fingerprint string
	PEM         string
}

type DBUser struct {
	Username         string
	Admin            bool
	MaxInstanceCount int
}

func InitDB(path string) {
	var err error
	DB, err = sql.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}
	if _, err := DB.Exec(initSQL); err != nil {
		panic(err)
	}
}

func ListPubkeys(username string) ([]DBPubKey, error) {
	rows, err := DB.Query("SELECT fingerprint, pubkey FROM pubkeys WHERE username = ?", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pubkeys []DBPubKey
	for rows.Next() {
		var pubkey DBPubKey
		if err = rows.Scan(&pubkey.Fingerprint, &pubkey.PEM); err != nil {
			return nil, err
		}
		pubkeys = append(pubkeys, pubkey)
	}
	return pubkeys, nil
}

func AddPubkey(username, pubkey string) error {
	hash := sha256.Sum256([]byte(pubkey))
	hexhash := hex.EncodeToString(hash[:])
	_, err := DB.Exec("INSERT INTO pubkeys (username, fingerprint, pubkey) VALUES (?, ?, ?)", username, hexhash, pubkey)
	return err
}

func DeletePubkey(username, fingerprint string) error {
	_, err := DB.Exec("DELETE FROM pubkeys WHERE username = ? AND SUBSTR(fingerprint, 1, ?) = ?", username, len(fingerprint), fingerprint)
	return err
}

func GetUser(username string) (DBUser, error) {
	var user DBUser
	err := DB.QueryRow("SELECT username, admin, max_instance_count FROM users WHERE username = ?", username).Scan(&user.Username, &user.Admin, &user.MaxInstanceCount)
	return user, err
}
