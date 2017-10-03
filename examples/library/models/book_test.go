package models

import "os"
import "fmt"
import "strings"
import "testing"
import _ "github.com/mattn/go-sqlite3"
import "database/sql"
import "github.com/franela/goblin"

func addBookRow(db *sql.DB, values ...[]string) error {
	for _, rowValues := range values {
		valueString := strings.Join(rowValues, ",")
		statement := fmt.Sprintf("insert into books (id,page_count,author_id,title) values(%s);", valueString)
		r, e := db.Exec(statement)

		if e != nil {
			return e
		}

		count, e := r.RowsAffected()

		if e != nil {
			return e
		}

		if count != 1 {
			return fmt.Errorf("no-rows-created")
		}
	}

	return nil
}

func Test_Book(t *testing.T) {
	dbFile := "./book-library.db"
	var db *sql.DB
	var bookStore *BookStore

	g := goblin.Goblin(t)

	defer os.Remove(dbFile)

	g.Describe("Book model & generated store", func() {

		g.Before(func() {
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

			bookFixtures := [][]string{}

			for i := 1; i < 150; i++ {
				id, pageCount, author := fmt.Sprintf("%d", i), fmt.Sprintf("%d", i*10), fmt.Sprintf("%d", (i*10)+1)
				title := fmt.Sprintf("'book-%d'", i)
				bookFixtures = append(bookFixtures, []string{id, pageCount, author, title})
			}

			g.Assert(addBookRow(db, bookFixtures...)).Equal(nil)
		})

		g.BeforeEach(func() {
			bookStore = &BookStore{DB: db}
		})

		g.After(func() {
			e := db.Close()
			g.Assert(e).Equal(nil)
			os.Remove(dbFile)
		})

		g.It("allows the consumer to search for books w/o a blueprint (default limit)", func() {
			books, e := bookStore.FindBooks(nil)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(10)
		})

		g.It("allows the consumer to search for books w/ blueprint (explicit offset)", func() {
			books, e := bookStore.FindBooks(&BookBlueprint{Offset: 1})
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(10)
			g.Assert(books[0].ID).Equal(2)
		})

		g.It("allows the consumer to search for books w/ blueprint (explicit offset)", func() {
			books, e := bookStore.FindBooks(&BookBlueprint{Limit: 20})
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(20)
		})

		g.It("allows the consumer to search for books w/ an exact match on the author id", func() {
			q := &BookBlueprint{AuthorID: []int{11, 21, 100}}
			books, e := bookStore.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(2)
		})

		g.It("allows the consumer to search for books w/ an exact match on the id", func() {
			q := &BookBlueprint{ID: []int{1, 2, 10e3}}
			books, e := bookStore.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(2)
		})

		g.It("allows the consumer to search for books w/ an exact match on the title", func() {
			q := &BookBlueprint{Title: []string{"book-1"}}
			books, e := bookStore.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(1)
		})

		g.It("allows the consumer to search for books w/ an author id range", func() {
			q := &BookBlueprint{AuthorIDRange: []int{0, 20}}
			books, e := bookStore.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(1)
		})

		g.It("allows the consumer to search for books w/ a page count range", func() {
			q := &BookBlueprint{PageCountRange: []int{0, 20}}
			books, e := bookStore.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(1)
		})

		g.It("allows the consumer to search for books w/ an id range", func() {
			q := &BookBlueprint{IDRange: []int{0, 3}}
			books, e := bookStore.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(2)
		})

		g.It("allows the consumer to search for books w/ multiple fields", func() {
			books, e := bookStore.FindBooks(&BookBlueprint{
				ID:             []int{1},
				PageCountRange: []int{0, 20},
			})

			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(1)
		})
	})
}
