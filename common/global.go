package common

import (
	"database/sql"
	"lxcpanel/lxc"
)

var (
	LxcClient *lxc.LXCClient
	DB        *sql.DB
)
