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
	var db *sql.DB
	var store *BookStore

	g := goblin.Goblin(t)

	dbFile := "./book-testing.db"

	g.Describe("Book model & generated store", func() {

		g.Before(func() {
			var e error
			db, e = loadDB(dbFile)
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
			store = &BookStore{DB: db}
		})

		g.After(func() {
			e := db.Close()
			g.Assert(e).Equal(nil)
			os.Remove(dbFile)
		})

		g.It("allows the consumer to search for books w/o a blueprint (default limit)", func() {
			books, e := store.FindBooks(nil)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(10)
		})

		g.It("allows the consumer to search for books w/ blueprint (explicit offset)", func() {
			books, e := store.FindBooks(&BookBlueprint{Offset: 1})
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(10)
			g.Assert(books[0].ID).Equal(2)
		})

		g.It("allows the consumer to search for books w/ blueprint (explicit offset)", func() {
			books, e := store.FindBooks(&BookBlueprint{Limit: 20})
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(20)
		})

		g.It("allows the consumer to search for books w/ an exact match on the author id", func() {
			q := &BookBlueprint{AuthorID: []int{11, 21, 100}}
			books, e := store.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(2)
		})

		g.It("allows the consumer to search for books w/ an exact match on the id", func() {
			q := &BookBlueprint{ID: []int{1, 2, 10e3}}
			books, e := store.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(2)
		})

		g.It("allows the consumer to search for books w/ an exact match on the title", func() {
			q := &BookBlueprint{Title: []string{"book-1"}}
			books, e := store.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(1)
		})

		g.It("allows the consumer to search for books w/ an author id range", func() {
			q := &BookBlueprint{AuthorIDRange: []int{0, 20}}
			books, e := store.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(1)
		})

		g.It("allows the consumer to search for books w/ a page count range", func() {
			q := &BookBlueprint{PageCountRange: []int{0, 20}}
			books, e := store.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(1)
		})

		g.It("allows the consumer to search for books w/ an id range", func() {
			q := &BookBlueprint{IDRange: []int{0, 3}}
			books, e := store.FindBooks(q)
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(2)
		})

		g.It("allows the consumer to search for books w/ multiple fields", func() {
			books, e := store.FindBooks(&BookBlueprint{
				ID:             []int{1},
				PageCountRange: []int{0, 20},
			})

			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(1)
		})

		g.It("allows the consumer to count books by blueprint", func() {
			count, e := store.CountBooks(&BookBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(count).Equal(2)
		})

		g.It("allows the consumer to select explicit book ids", func() {
			results, e := store.SelectIDs(&BookBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results[0]).Equal(1)
			g.Assert(results[1]).Equal(2)
		})

		g.It("allows the consumer to select explicit book titles", func() {
			results, e := store.SelectTitles(&BookBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results[0]).Equal("book-1")
			g.Assert(results[1]).Equal("book-2")
		})

		g.It("allows the consumer to select explicit author ids", func() {
			results, e := store.SelectAuthorIDs(&BookBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results[0]).Equal(11)
			g.Assert(results[1]).Equal(21)
		})

		g.It("allows the consumer to select explicit page counts", func() {
			results, e := store.SelectPageCounts(&BookBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results[0]).Equal(10)
			g.Assert(results[1]).Equal(20)
		})

		g.It("returns 0 recods when update is called with nil updates and nil clause", func() {
			count, e := store.UpdateBooks(nil, nil)
			g.Assert(e).Equal(nil)
			g.Assert(count).Equal(0)
		})
	})
}
