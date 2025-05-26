package main

import "database/sql"

type Level struct {
	Id int64
	Name string
	Description string
	UserId int64
	Data []byte
}

type LevelItem struct {
	Id int64
	Name string
	UserName string
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
		description text,
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
	err := row.Scan(&result.Id, &result.Name, &result.Description, &result.UserId, &result.Data)
	return result, err
}

func dbCreateLevel(db *sql.DB, name string, description string, userId int64, data []byte) (Level, int64, error) {
	level := Level { Id: -1 }
	user := dbFindUserWithId(db, userId)

	if user.id == -1 {
		return level, -1, ErrUserAlreadyExists
	}

	statement := `
	INSERT INTO Levels (name, description, userId, data) 
	VALUES (?, ?, ?, ?) RETURNING id, name, description, userId, data
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare statement. SQL: ", statement) {
		return level, -1, ErrUserCouldNotCreate
	}
	defer stmt.Close()
	
	row := stmt.QueryRow(name, description, userId, data)
	level, err = dbScanLevel(row)

	return level, level.Id, err
}

func dbGetLevel(db *sql.DB, id int64) Level {
	statement := `
	SELECT id, name, description, userId, data
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

func dbUpdateLevel(db *sql.DB, levelId int64, description string, data []byte) error {
	statement := `
	UPDATE Levels SET description=?, data=?
	WHERE id=?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare statement. SQL: ", statement) {
		return err
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(description, data, levelId)

	return err
}

func dbScanLevelItem(rows *sql.Rows) (LevelItem, error) {
	result := LevelItem { Id: -1 }
	err := rows.Scan(&result.Id, &result.Name, &result.UserName)
	return result, err
}

type LevelSort int8
const (
	LevelSortRecent LevelSort = 1
	LevelSortBest LevelSort = 2
)
func dbGetLevels(db *sql.DB, sorting LevelSort, page int, limit int) ([]LevelItem, int) {
	statement := `
	SELECT lvl.id, lvl.name, usr.name
	FROM Levels lvl INNER JOIN Users usr ON usr.id = lvl.userId
	LIMIT ? OFFSET ?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare find level with id statement. SQL: ", statement) {
		return []LevelItem{}, 0
	}
	defer stmt.Close()

	rows, err := stmt.Query(limit, page * limit)
	if didFail(err, "could not query. SQL: ", statement) {
		return []LevelItem{}, 0
	}

	result := []LevelItem{}
	count := 0
	for rows.Next() {
		item, _ := dbScanLevelItem(rows)
		result = append(result, item)
		count += 1
	}
	rows.Close()
	
	new_page := page
	if count > 0 {
		new_page += 1
	}

	return result, new_page
}