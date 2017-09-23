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
			os.Remove(dbFile)
		})

		g.It("allows the consumer to search for books w/o a blueprint", func() {
			_, e := bookStore.FindBooks(nil)
			g.Assert(e).Equal(nil)
		})

		g.It("allows the consumer to search for books w/ a blueprint & multiple fields", func() {
			q := &BookBlueprint{
				IDRange: []int{0, 100},
			}

			r, dbErr := db.Exec(`
			insert into books (id,page_count,author_id,title) values(1,50,1,'hello world');
			`)

			g.Assert(dbErr).Equal(nil)

			count, _ := r.RowsAffected()
			g.Assert(count).Equal(1)

			books, e := bookStore.FindBooks(q)

			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(1)
		})

		g.It("allows the consumer to search for books", func() {
			_, e := bookStore.FindBooks(&BookBlueprint{
				IDRange: []int{4, 5},
			})
			g.Assert(e).Equal(nil)
		})

	})
}
