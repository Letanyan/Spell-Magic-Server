package main

import (
	"database/sql"
)

const (
    isDebug = true
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