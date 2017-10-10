package models

import "io"
import "os"
import "bytes"
import _ "github.com/mattn/go-sqlite3"
import "database/sql"

func loadDB(dbFile string) (*sql.DB, error) {
	db, e := sql.Open("sqlite3", dbFile)

	if e != nil {
		return nil, e
	}

	file, e := os.Open("../schema.sql")

	if e != nil {
		return nil, e
	}

	defer file.Close()

	buffer := new(bytes.Buffer)

	if _, e := io.Copy(buffer, file); e != nil {
		return nil, e
	}

	if _, e := db.Exec(buffer.String()); e != nil {
		return nil, e
	}

	return db, nil
}
