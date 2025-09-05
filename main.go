package main

import (
	"database/sql"
)

const (
    isDebug = false
)

var (
    mainDB *sql.DB
)

func main() {
    mainDB = dbConnect()
    defer mainDB.Close()
    dbInit(mainDB)

    serverInit()
}