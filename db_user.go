package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"math/rand/v2"
	"strconv"
)

type UserRole int8
const (
	UserRoleDeveloper UserRole = 0
	UserRoleCallCenterManager UserRole = 1
	UserRoleCallCenterAgent UserRole = 2
)

type UserError int8
const (
	ErrUserDoesNotExist UserError = 1
	ErrUserCouldNotCreate UserError = 2
	ErrUserIncorrectPassword UserError = 3
	ErrUserFailedSignIn UserError = 4
	ErrUserAlreadyExists UserError = 5
)
func (err UserError) Error() string {
	switch err {
	case 1: return "User does not exist"
	case 2: return "Could not create user"
	case 3: return "Incorrect password"
	case 4: return "Failed Sign-In"
	case 5: return "User already exists"
	}
	return ""
}


type User struct {
	id int64
	name string
	password string
}

func dbCreateUserTable(db *sql.DB) {
	statement := `
	CREATE TABLE IF NOT EXISTS Users (
		id integer not null primary key,
		name text,
		password text
	);
	`
	_, err := db.Exec(statement)
	if didFail(err, "could not create table 'Users'. SQL: ", statement) {
		return
	}

	statement = `
	CREATE TABLE IF NOT EXISTS UsersAuth (
		session_token text not null PRIMARY KEY,
		user_id integer not null
	);
	`
	_, err = db.Exec(statement)
	if didFail(err, "could not create table 'UsersAuth'. SQL: ", statement) {
		return
	}
}

func hashText(password string) string {
	h := sha256.New()
	h.Write([]byte("[_" + password + "_]"))
	result := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return result
}

func generateRandomString(length int) string {
	tokens := "1234567890qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM"
	result := ""
	for i := 0; i < int(length); i++ {
		r := rand.Int32N(int32(len(tokens)))
		result += string(tokens[r])
	}
	return result
}

func dbCreateUser(db *sql.DB, name string, password string) (User, string, error) {
	user := User { id: -1 }

	statement := `
	INSERT INTO Users (name, password) 
	VALUES (?, ?)
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare statement. SQL: ", statement) {
		return user, "", ErrUserCouldNotCreate
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(name, hashText(password))
	if didFail(err, "could not execute prepared statement. SQL: ", statement) {
		return user, "", ErrUserCouldNotCreate
	}

	return dbSignInUser(db, name, password)
}

func dbScanUser(row *sql.Row) (User, error) {
	result := User { id: -1 }
	err := row.Scan(&result.id, &result.name, &result.password)
	return result, err
}

func dbFindUserWithName(db *sql.DB, name string) (User, error) {
	statement := `
	SELECT id, name, password
	FROM Users 
	WHERE name=?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare find user with email statement. SQL: ", statement) {
		return User { id: -1 }, ErrUserDoesNotExist
	}
	defer stmt.Close()

	row := stmt.QueryRow(name)
	return dbScanUser(row)
}

func dbUserWithNameExists(db *sql.DB, name string) bool {
	statement := `
	SELECT name
	FROM Users 
	WHERE name=?
	`
	stmt, err := db.Prepare(statement)
	if err != nil {
		return false
	}
	defer stmt.Close()

	row := stmt.QueryRow(name)
	user, err := dbScanUser(row)
	if err != nil {
		return false
	}

	return user.id != -1
}

func dbFindUserWithId(db *sql.DB, id int64) User {
	statement := `
	SELECT id, name, password
	FROM Users 
	WHERE id=?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare find user with id statement. SQL: ", statement) {
		return User { id: -1 }
	}
	defer stmt.Close()

	row := stmt.QueryRow(id)
	user, err := dbScanUser(row)
	if didFail(err) {
		return User {id: -1}
	} else {
		return user
	}
}

func dbRemoveUserAuth(db *sql.DB, id int64) {
	statement := `
	DELETE FROM UsersAuth WHERE user_id=?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare insert into auth statement. SQL: ", statement) {
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	if didFail(err, "could not execute prepared statement. SQL: ", statement) {
		return
	}
}

func dbSignInUser(db *sql.DB, name string, password string) (User, string, error) {
	user, err := dbFindUserWithName(db, name)
	if err != nil {
		return user, "", ErrUserFailedSignIn
	}

	if user.password != hashText(password) {
		return user, "", ErrUserIncorrectPassword
	}

	dbRemoveUserAuth(db, user.id)

	statement := `
	INSERT INTO UsersAuth (session_token, user_id)
	VALUES (?, ?)
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare insert into auth statement. SQL: ", statement) {
		return user, "", ErrUserFailedSignIn
	}
	defer stmt.Close()

	left_amount := rand.IntN(24)
	right_amount := 128 - left_amount
	session_token := generateRandomString(left_amount) + strconv.FormatInt(user.id, 36) + generateRandomString(right_amount)
	_, err = stmt.Exec(session_token, user.id)
	if didFail(err, "could not execute prepared statement. SQL: ", statement) {
		return user, "", ErrUserFailedSignIn
	}

	return user, session_token, nil
}

func dbDoesUserHaveASession(db *sql.DB, session_token string) User {
	statement := `
	SELECT user_id
	FROM UsersAuth
	WHERE session_token=?
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare find user session token. SQL: ", statement) {
		return User{ id: -1 }
	}
	defer stmt.Close()

	row := stmt.QueryRow(session_token)
	var id int64
	err = row.Scan(&id)
	if err == nil {
		return dbFindUserWithId(db, id)
	} else {
		return User{ id: -1 }
	}
}

func dbSignOutUserSession(db *sql.DB, sessionToken string) {
	statement := `
	DELETE
	FROM UsersAuth
	WHERE session_token=?;
	`
	stmt, err := db.Prepare(statement)
	if didFail(err, "could not prepare find user session token. SQL: ", statement) {
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(sessionToken)
	if didFail(err, "could not exec query. SQL: ", statement) {
		return
	}
}