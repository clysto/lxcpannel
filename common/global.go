package common

import (
	"database/sql"
	"lxcpanel/lxc"
)

var (
	Client *lxc.LXCClient
	DB     *sql.DB
)
