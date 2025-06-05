package main

import (
	"database/sql"
	"fmt"
)

type Level struct {
	Id int64
	Name string
	Description string
	UserId int64
	UserVote int64
	UserPlaytime float64
	Data []byte
}

type LevelItem struct {
	Id int64
	Name string
	UserName string
	Votes int64
	Playtime float64
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
		id integer NOT NULL PRIMARY KEY,
		name text DEFAULT '',
		description text DEFAULT '',
		userId integer NOT NULL,
		data blob
	);
	`
	_, err := db.Exec(statement)
	if didFail(err, "could not create table 'Levels'. SQL: ", statement) {
		return
	}

	statement = `
	CREATE TABLE IF NOT EXISTS LevelUserData (
		userId integer NOT NULL,
		levelId integer NOT NULL,
		vote DEFAULT 0,
		playtime real DEFAULT 0.0,
		startPlay integer DEFAULT (unixEpoch()),
		PRIMARY KEY (userId, levelId)
	);
	`
	_, err = db.Exec(statement)
	if didFail(err, "could not create table 'LevelUserData'. SQL: ", statement) {
		return
	}
}

func dbScanLevel(row *sql.Row) (Level, error) {
	result := Level { Id: -1 }
	err := row.Scan(&result.Id, &result.Name, &result.Description, &result.UserId, &result.UserVote, &result.UserPlaytime, &result.Data)
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

func dbGetLevel(db *sql.DB, id int64, userId int64) Level {
	statement := `
	SELECT lvl.id, lvl.name, lvl.description, lvl.userId, IFNULL(data.vote, 0), IFNULL(data.playtime, 0.0), lvl.data
	FROM Levels lvl LEFT JOIN LevelUserData data ON data.levelId = lvl.id AND data.userId = ?
	WHERE lvl.id=?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare find level with id statement. SQL: ", statement) {
		return Level { Id: -1 }
	}
	defer stmt.Close()

	row := stmt.QueryRow(userId, id)
	level, err := dbScanLevel(row)
	if didFail(err) {
		return Level {Id: -1}
	} else {
		return level
	}
}

func dbUpdateLevel(db *sql.DB, user User, levelId int64, description string, data []byte) error {
	statement := `
	UPDATE Levels SET description=?, data=?
	WHERE id=? AND userId=?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare statement. SQL: ", statement) {
		return err
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(description, data, levelId, user.id)

	return err
}

func dbScanLevelItem(rows *sql.Rows) (LevelItem, error) {
	result := LevelItem { Id: -1 }
	err := rows.Scan(&result.Id, &result.Name, &result.UserName, &result.Votes, &result.Playtime)
	return result, err
}

type LevelSort int8
const (
	LevelSortRecent LevelSort = 1
	LevelSortBest LevelSort = 2
)
func dbGetLevelItems(rows *sql.Rows) ([]LevelItem, int) {
	result := []LevelItem{}
	count := 0
	for rows.Next() {
		item, _ := dbScanLevelItem(rows)
		result = append(result, item)
		count += 1
	}
	rows.Close()
	
	increment := 0
	if count > 0 {
		increment += 1
	}

	return result, increment
}

func dbGetLevels(db *sql.DB, sorting LevelSort, page int, limit int) ([]LevelItem, int) {
	statement := `
	SELECT lvl.id, lvl.name, usr.name, SUM(data.vote), SUM(data.playtime)
	FROM Levels lvl 
		INNER JOIN Users usr ON usr.id = lvl.userId
		FULL JOIN LevelUserData data ON lvl.id = data.levelId
	GROUP BY lvl.id
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

	result, pageIncrement := dbGetLevelItems(rows) 
	new_page := page + pageIncrement
	return result, new_page
}

func dbGetLevelsWithName(db *sql.DB, search string, sorting LevelSort, page int, limit int) ([]LevelItem, int) {
	statement := `
	SELECT lvl.id, lvl.name, usr.name, SUM(data.vote), SUM(data.playtime)
	FROM Levels lvl 
		INNER JOIN Users usr ON usr.id = lvl.userId
		FULL JOIN LevelUserData data ON lvl.id = data.levelId
	GROUP BY lvl.id
	WHERE lvl.name LIKE ?
	LIMIT ? OFFSET ?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare find level with id statement. SQL: ", statement) {
		return []LevelItem{}, 0
	}
	defer stmt.Close()
	
	rows, err := stmt.Query("%" + search + "%", limit, page * limit)
	if didFail(err, "could not query. SQL: ", statement) {
		return []LevelItem{}, 0
	}
	
	result, pageIncrement := dbGetLevelItems(rows) 
	new_page := page + pageIncrement
	for _, item := range result {
		fmt.Println(item)
	}
	return result, new_page
}

func dbGetLevelsFromUser(db *sql.DB, search string, sorting LevelSort, page int, limit int) ([]LevelItem, int) {
	statement := `
	SELECT lvl.id, lvl.name, usr.name, SUM(data.vote), SUM(data.playtime)
	FROM Levels lvl 
		INNER JOIN Users usr ON usr.id = lvl.userId
		FULL JOIN LevelUserData data ON lvl.id = data.levelId
	GROUP BY lvl.id
	WHERE usr.name LIKE ?
	LIMIT ? OFFSET ?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare find level with id statement. SQL: ", statement) {
		return []LevelItem{}, 0
	}
	defer stmt.Close()

	rows, err := stmt.Query("%" + search + "%", limit, page * limit)
	if didFail(err, "could not query. SQL: ", statement) {
		return []LevelItem{}, 0
	}

	result, pageIncrement := dbGetLevelItems(rows) 
	new_page := page + pageIncrement
	return result, new_page
}

func dbCreateLevelUserData(db *sql.DB, levelId int64, userId int64) error {
	statement := `
	INSERT OR IGNORE INTO LevelUserData (levelId, userId, vote, playtime) 
	VALUES (?, ?, 0, 0.0)
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare statement. SQL: ", statement) {
		return ErrUserCouldNotCreate
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(levelId, userId)

	return err
}

func dbUpdateLevelUserData(db *sql.DB, levelId int64, userId int64, vote int64) error {
	statement := `
	UPDATE LevelUserData SET vote=?
	WHERE levelId = ? AND userId = ?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare statement. SQL: ", statement) {
		return err
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(vote, levelId, userId)

	return err
}

func dbBeginLevelUserDataPlaytime(db *sql.DB, levelId int64, userId int64) error {
	statement := `
	UPDATE LevelUserData SET startPlay=unixEpoch()
	WHERE levelId = ? AND userId = ?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare statement. SQL: ", statement) {
		return err
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(levelId, userId)

	return err
}

func dbSaveLevelUserDataPlaytime(db *sql.DB, levelId int64, userId int64) error {
	statement := `
	UPDATE LevelUserData SET playtime=playtime+unixEpoch()-startPlay, startPlay=unixEpoch()
	WHERE levelId = ? AND userId = ?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare statement. SQL: ", statement) {
		return err
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(levelId, userId)

	return err
}