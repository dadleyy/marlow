package models

import "os"
import "testing"
import _ "github.com/mattn/go-sqlite3"
import "database/sql"
import "github.com/franela/goblin"

const (
	dbFile = "./library.db"
)

func Test_Book(t *testing.T) {
	var db *sql.DB
	var bookStore *BookStore

	g := goblin.Goblin(t)

	os.Remove(dbFile)
	defer os.Remove(dbFile)

	g.Describe("Book model & generated store", func() {

		g.BeforeEach(func() {
			var connError error
			db, connError = sql.Open("sqlite3", dbFile)
			g.Assert(connError).Equal(nil)

			sqlStmt := `
				create table books (
					id integer not null primary key,
					title text,
					author_id integer not null,
					page_count integer not null
				);
				delete from books;
			`

			_, e := db.Exec(sqlStmt)
			g.Assert(e).Equal(nil)

			bookStore = &BookStore{DB: db}
		})

		g.AfterEach(func() {
			e := db.Close()
			g.Assert(e).Equal(nil)
		})

		g.It("allows the consumer to search for books", func() {
			_, e := bookStore.FindBooks(nil)
			g.Assert(e).Equal(nil)
		})

	})
}
