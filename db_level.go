package main

import "database/sql"

type Level struct {
	Id int64
	Name string
	UserId int64
	Data []byte
}

type LevelError int8
const (
	ErrLevelCouldNotCreate LevelError = 1
)
func (err LevelError) Error() string {
	switch err {
	case 1: return "Could not create level"
	}
	return ""
}

func dbCreateLevelsTable(db *sql.DB) {
	statement := `
	CREATE TABLE IF NOT EXISTS Levels (
		id integer not null primary key,
		name text,
		userId integer,
		data blob
	);
	`
	_, err := db.Exec(statement)
	if didFail(err, "could not create table 'Levels'. SQL: ", statement) {
		return
	}
}

func dbScanLevel(row *sql.Row) (Level, error) {
	result := Level { Id: -1 }
	err := row.Scan(&result.Id, &result.Name, &result.UserId, &result.Data)
	return result, err
}

func dbCreateLevel(db *sql.DB, name string, userId int64, data []byte) (Level, int64, error) {
	level := Level { Id: -1 }
	user := dbFindUserWithId(db, userId)

	if user.id == -1 {
		return level, -1, ErrUserAlreadyExists
	}

	statement := `
	INSERT INTO Levels (name, userId, data) 
	VALUES (?, ?, ?) RETURNING id, name, userId, data
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare statement. SQL: ", statement) {
		return level, -1, ErrUserCouldNotCreate
	}
	defer stmt.Close()
	
	row := stmt.QueryRow(name, userId, data)
	level, err = dbScanLevel(row)

	return level, level.Id, err
}

func dbGetLevel(db *sql.DB, id int64) Level {
	statement := `
	SELECT id, name, userId, data
	FROM Levels
	WHERE id=?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare find level with id statement. SQL: ", statement) {
		return Level { Id: -1 }
	}
	defer stmt.Close()

	row := stmt.QueryRow(id)
	level, err := dbScanLevel(row)
	if didFail(err) {
		return Level {Id: -1}
	} else {
		return level
	}
}