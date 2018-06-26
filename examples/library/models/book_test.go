package models

import "os"
import "io"
import "fmt"
import "bytes"
import "strings"
import "testing"
import _ "github.com/mattn/go-sqlite3"
import "database/sql"
import "github.com/franela/goblin"

func addBookRow(db *sql.DB, values ...[]string) error {
	for _, rowValues := range values {
		valueString := strings.Join(rowValues, ",")
		statement := fmt.Sprintf("insert into books (year_published,author,title) values(%s);", valueString)
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
	var store BookStore
	var queryLog io.Writer

	g := goblin.Goblin(t)
	testBookCount := 150

	dbFile := "./book-testing.db"

	g.Describe("Book Blueprint test suite", func() {
		g.It("returns value placeholders for TitleLike", func() {
			str := fmt.Sprintf("%s", &BookBlueprint{TitleLike: []string{"b"}})
			g.Assert(str).Equal("WHERE books.title LIKE ?")
		})

		g.It("allows inclusive (OR) blueprints", func() {
			str := fmt.Sprintf("%s", &BookBlueprint{
				TitleLike: []string{"b", "c"},
				IDRange:   []int{1, 2},
				Inclusive: true,
			})
			expected := "WHERE (books.system_id > ? AND books.system_id < ?) OR books.title LIKE ? OR books.title LIKE ?"
			g.Assert(str).Equal(expected)
		})
	})

	g.Describe("Book model & generated store", func() {

		g.Before(func() {
			var e error
			db, e = loadDB(dbFile)
			g.Assert(e).Equal(nil)

			bookFixtures := [][]string{}

			for i := 1; i < testBookCount+1; i++ {
				yearPublished, author := fmt.Sprintf("200%d", i), fmt.Sprintf("%d", (i*10)+1)
				title := fmt.Sprintf("'book-%d'", i)
				bookFixtures = append(bookFixtures, []string{yearPublished, author, title})
			}

			g.Assert(addBookRow(db, bookFixtures...)).Equal(nil)
		})

		g.BeforeEach(func() {
			queryLog = new(bytes.Buffer)
			store = NewBookStore(db, queryLog)
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

		g.It("allows the consumer to search for books w/ an exact match on the series id (not null)", func() {
			books, e := store.FindBooks(&BookBlueprint{SeriesID: []sql.NullInt64{}})
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(0)
		})

		g.It("allows the consumer to search by title like", func() {
			count, e := store.CountBooks(&BookBlueprint{TitleLike: []string{"%book-%"}})
			g.Assert(e).Equal(nil)
			g.Assert(count).Equal(testBookCount)
		})

		g.It("allows the consumer to select series id", func() {
			ids, e := store.SelectBookSeriesIDs(&BookBlueprint{ID: []int{1}})
			g.Assert(e).Equal(nil)
			g.Assert(ids[0].Valid).Equal(false)
		})

		g.It("allows the consumer to search for books w/ an exact match on the series id (null)", func() {
			books, e := store.FindBooks(&BookBlueprint{SeriesID: []sql.NullInt64{
				{Valid: false},
			}})
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(10)
		})

		g.It("allows the consumer to search for books w/ an exact match on the series id (valid)", func() {
			books, e := store.FindBooks(&BookBlueprint{SeriesID: []sql.NullInt64{
				{Valid: true, Int64: 10},
			}})
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(0)
		})

		g.It("allows the consumer to search for books w/ an exact match on the year published", func() {
			books, e := store.FindBooks(&BookBlueprint{YearPublished: []int{2001, 2002}})
			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(2)
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

		g.It("allows the consumer to search for books w/ a year published range", func() {
			q := &BookBlueprint{YearPublishedRange: []int{2000, 2002}}
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
				ID:                 []int{1},
				YearPublishedRange: []int{2000, 2003},
			})

			g.Assert(e).Equal(nil)
			g.Assert(len(books)).Equal(1)
		})

		g.Describe("store.CountBooks", func() {

			g.It("allows the consumer to count books with nil blueprint", func() {
				count, e := store.CountBooks(nil)
				g.Assert(e).Equal(nil)
				g.Assert(count).Equal(testBookCount)
			})

			g.It("allows the consumer to count books by blueprint", func() {
				count, e := store.CountBooks(&BookBlueprint{
					ID: []int{1, 2},
				})
				g.Assert(e).Equal(nil)
				g.Assert(count).Equal(2)
			})

		})

		g.It("allows the consumer to select explicit book ids", func() {
			results, e := store.SelectBookIDs(&BookBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results[0]).Equal(1)
			g.Assert(results[1]).Equal(2)
		})

		g.It("returns source struct instances that support source method calls", func() {
			results, e := store.FindBooks(&BookBlueprint{
				ID: []int{1},
			})
			g.Assert(e).Equal(nil)
			title := fmt.Sprintf("%s", results[0])
			g.Assert(title).Equal("book-1 (published in 2001)")
		})

		g.It("allows the consumer to select explicit book titles", func() {
			results, e := store.SelectBookTitles(&BookBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results[0]).Equal("book-1")
			g.Assert(results[1]).Equal("book-2")
		})

		g.It("allows the consumer to select explicit author ids", func() {
			results, e := store.SelectBookAuthorIDs(&BookBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results[0]).Equal(11)
			g.Assert(results[1]).Equal(21)
		})

		g.It("allows the consumer to update the book title", func() {
			results, e := store.UpdateBookTitle("Marlow the puppy", &BookBlueprint{
				ID: []int{1},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results).Equal(int64(1))
		})

		g.It("allows the consumer to update the author id", func() {
			results, e := store.UpdateBookAuthorID(2001, &BookBlueprint{
				ID: []int{1},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results).Equal(int64(1))
		})

		g.It("allows the consumer to update the book year published", func() {
			results, e := store.UpdateBookYearPublished(10, &BookBlueprint{
				ID: []int{10},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results).Equal(int64(1))
		})

		g.It("allows the consumer to update the book series id (valid: false)", func() {
			results, e := store.UpdateBookSeriesID(&sql.NullInt64{
				Valid: false,
			}, &BookBlueprint{
				ID: []int{10},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results).Equal(int64(1))
		})

		g.It("allows the consumer to update the book series id (non-nil)", func() {
			results, e := store.UpdateBookSeriesID(&sql.NullInt64{
				Valid: true,
				Int64: 10,
			}, &BookBlueprint{
				ID: []int{10},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results).Equal(int64(1))
		})

		g.It("allows the consumer to update the book series id (nil)", func() {
			results, e := store.UpdateBookSeriesID(nil, &BookBlueprint{
				ID: []int{10},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results).Equal(int64(1))
		})

		g.It("allows the consumer to update the book id", func() {
			results, e := store.UpdateBookID(2001, &BookBlueprint{
				ID: []int{10},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results).Equal(int64(1))
		})

		g.It("allows the consumer to select explicit year published", func() {
			results, e := store.SelectBookYearPublisheds(&BookBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(results[0]).Equal(2001)
			g.Assert(results[1]).Equal(2002)
		})

		g.Describe("DeleteBooks", func() {

			g.It("returns an error and a negative number with an empty blueprint", func() {
				c, e := store.DeleteBooks(&BookBlueprint{})
				g.Assert(e == nil).Equal(false)
				g.Assert(c).Equal(int64(-1))
			})

			g.It("returns an error and a negative number without a blueprint", func() {
				c, e := store.DeleteBooks(nil)
				g.Assert(e == nil).Equal(false)
				g.Assert(c).Equal(int64(-1))
			})

			g.It("successfully returns 0 if no books were found to delete", func() {
				deleted, e := store.DeleteBooks(&BookBlueprint{ID: []int{-1000}})
				g.Assert(e).Equal(nil)
				g.Assert(deleted).Equal(int64(0))
			})

			g.It("successfully deletes the records found by the blueprint", func() {
				deleted, e := store.DeleteBooks(&BookBlueprint{ID: []int{13}})
				g.Assert(e).Equal(nil)
				g.Assert(deleted).Equal(int64(1))
				found, e := store.CountBooks(&BookBlueprint{ID: []int{13}})
				g.Assert(e).Equal(nil)
				g.Assert(found).Equal(0)
			})
		})

		g.Describe("CreateBooks", func() {
			g.It("returns immediately with 0 if no authors", func() {
				s, e := store.CreateBooks()
				g.Assert(e).Equal(nil)
				g.Assert(s).Equal(int64(0))
			})

			g.It("returns the number of authors created", func() {
				s, e := store.CreateBooks([]Book{
					{Title: "Lord of the Rings", YearPublished: 2018, AuthorID: 1},
					{Title: "Harry Potter", YearPublished: 2010, AuthorID: 2},
				}...)
				g.Assert(e).Equal(nil)

				p, e := store.FindBooks(&BookBlueprint{ID: []int{int(s)}})
				g.Assert(e).Equal(nil)
				g.Assert(p[0].Title).Equal("Harry Potter")

				found, e := store.FindBooks(&BookBlueprint{Title: []string{"Harry Potter"}})
				g.Assert(e).Equal(nil)
				g.Assert(len(found)).Equal(1)
				g.Assert(found[0].ID > 0).Equal(true)
			})
		})

		g.Describe("findAuthors", func() {
			g.It("successfully escapes single quote characters during searches on name", func() {
				name := "mr astley's blueberries"
				_, e := store.CreateBooks(Book{Title: name})
				g.Assert(e).Equal(nil)

				bp := &BookBlueprint{
					Title: []string{name},
				}

				c, e := store.CountBooks(bp)
				g.Assert(e).Equal(nil)
				g.Assert(c).Equal(1)

				_, e = store.DeleteBooks(bp)
				g.Assert(e).Equal(nil)
			})
		})
	})
}
