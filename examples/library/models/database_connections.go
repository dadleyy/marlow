package models

import "io"
import "fmt"
import "database/sql"
import _ "github.com/lib/pq"
import _ "github.com/mattn/go-sqlite3"
import "github.com/dadleyy/marlow/examples/library/data"

// marlow:ignore

const (
	psqlConnectionString = "user=%s dbname=%s port=%s sslmode=disable password=%s"
)

type DatabaseConnections struct {
	Config   *DatabaseConfig
	postgres *sql.DB
	sqlite   *sql.DB
}

func (db *DatabaseConnections) Initialize() error {
	sqlite, e := sql.Open("sqlite3", db.Config.SQLite.Filename)

	if e != nil {
		return e
	}

	schema, e := data.Asset("data/sqlite.sql")

	if e != nil {
		return e
	}

	r, e := sqlite.Exec(string(schema))
	if r == nil || e != nil {
		return fmt.Errorf("unable to load sqlite schema (e %v)", e)
	}

	db.sqlite = sqlite

	constr := fmt.Sprintf(
		psqlConnectionString,
		db.Config.Postgres.Username,
		db.Config.Postgres.Database,
		db.Config.Postgres.Port,
		db.Config.Postgres.Password,
	)

	postgres, e := sql.Open("postgres", constr)

	if e != nil {
		return fmt.Errorf("postgres connection error (e %s)", e.Error())
	}

	schema, e = data.Asset("data/postgres.sql")

	if e != nil {
		return fmt.Errorf("unable to load postgres schema (e %s)", e.Error())
	}

	if _, e := postgres.Exec(string(schema)); e != nil {
		return fmt.Errorf("unable to seed postgres db (e %s)", e.Error())
	}

	db.postgres = postgres

	return nil
}

func (db *DatabaseConnections) Stores(logger io.Writer) *Stores {
	return &Stores{
		Books:   NewBookStore(db.sqlite, nil),
		Authors: NewAuthorStore(db.sqlite, nil),
		Genres:  NewGenreStore(db.postgres, nil),
	}
}

func (db *DatabaseConnections) Close() error {
	dbs := []*sql.DB{db.sqlite, db.postgres}
	var e error

	for _, d := range dbs {
		if d == nil {
			continue
		}

		e = d.Close()
	}

	return e
}
