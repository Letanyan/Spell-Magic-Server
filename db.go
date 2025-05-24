package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type SqlError int8
const (
	ErrSQLFailedPrepareStatement SqlError = 1
	ErrSQLFailedExecuteStatement SqlError = 2
	ErrSQLFailedBeginTransaction SqlError = 3
	ErrSQLFailedCommitTransaction SqlError = 4
)
func (err SqlError) Error() string {
	switch err {
	case 1: return "Could not prepare statement"
	case 2: return "Could not execute statement"
	case 3: return "Could not begin transaction"
	case 4: return "Could not commit transaction"
	}
	return ""
}


func dbConnect() *sql.DB {
	db, err := sql.Open("sqlite3", "main.db")
	if didFail(err, "could not connect to 'main.db'") {
		log.Fatal(err)
	}
	err = db.Ping()
	if didFail(err, "could not connect to database") {
        log.Fatal(err)
    }
	return db
}

func dbInit(db *sql.DB) {
	dbCreateUserTable(db)
	dbCreateLevelsTable(db)
}

