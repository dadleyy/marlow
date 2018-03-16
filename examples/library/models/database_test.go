package models

import _ "github.com/mattn/go-sqlite3"
import "database/sql"
import "github.com/dadleyy/marlow/examples/library/data"

func loadDB(dbFile string) (*sql.DB, error) {
	db, e := sql.Open("sqlite3", dbFile)

	if e != nil {
		return nil, e
	}

	buffer, e := data.Asset("data/sqlite.sql")

	if e != nil {
		return nil, e
	}

	if _, e := db.Exec(string(buffer)); e != nil {
		return nil, e
	}

	return db, nil
}
