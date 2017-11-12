package models

import "os"
import "fmt"
import "strings"
import "testing"
import _ "github.com/mattn/go-sqlite3"
import "database/sql"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/marlow"

func addAuthorRow(db *sql.DB, values ...[]string) error {
	for _, rowValues := range values {
		valueString := strings.Join(rowValues, ",")
		statement := fmt.Sprintf("insert into authors (id,name) values(%s);", valueString)
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

func Test_Author(t *testing.T) {
	g := goblin.Goblin(t)
	var db *sql.DB
	var store AuthorStore

	dbFile := "author-testing.db"
	generatedAuthorCount := 150

	g.Describe("AuthorBlueprint test suite", func() {

		g.It("results in empty string w/o values", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{})
			g.Assert(r).Equal("")
		})

		g.It("supports int values on sql.NullInt64 fields", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{
				UniversityID: []sql.NullInt64{
					{Int64: 10, Valid: true},
				},
			})
			g.Assert(r).Equal("WHERE authors.university_id IN (?)")
		})

		g.It("supports NOT NULL selection if present but empty on sql.NullInt64 fields", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{
				UniversityID: []sql.NullInt64{},
			})
			g.Assert(r).Equal("WHERE authors.university_id NOT NULL")
		})

		g.It("supports null values on sql.NullInt64 fields", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{
				UniversityID: []sql.NullInt64{
					{Valid: false},
				},
			})
			g.Assert(r).Equal("WHERE authors.university_id IS NULL")
		})

		g.It("supports range on ID column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{
				IDRange: []int{1, 2},
			})

			g.Assert(r).Equal("WHERE authors.id > ? AND authors.id < ?")
		})

		g.It("supports 'IN' on ID column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{ID: []int{1, 2, 3}})
			g.Assert(r).Equal("WHERE authors.id IN (?,?,?)")
		})

		g.It("supports a combination of range and 'IN' on ID column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{
				ID:      []int{1, 2, 3},
				IDRange: []int{1, 4},
			})

			g.Assert(r).Equal("WHERE authors.id IN (?,?,?) AND authors.id > ? AND authors.id < ?")
		})

	})

	g.Describe("Author model & generated store test suite", func() {

		g.Before(func() {
			var e error
			db, e = loadDB(dbFile)
			g.Assert(e).Equal(nil)

			authors := [][]string{}

			for i := 1; i < generatedAuthorCount; i++ {
				id, name := fmt.Sprintf("%d", i), fmt.Sprintf("'author-%d'", (i*10)+1)
				authors = append(authors, []string{id, name})
			}

			g.Assert(addAuthorRow(db, authors...)).Equal(nil)

			_, e = db.Exec("insert into authors (id,name,university_id) values(1337,'learned author',10);")
			g.Assert(e).Equal(nil)
			_, e = db.Exec("insert into authors (id,name,university_id) values(1338,'other author',null);")
			g.Assert(e).Equal(nil)
		})

		g.BeforeEach(func() {
			store = NewAuthorStore(db)
		})

		g.After(func() {
			e := db.Close()
			g.Assert(e).Equal(nil)
			os.Remove(dbFile)
		})

		g.It("allows the consumer to search for authors w/o (default limit)", func() {
			authors, e := store.FindAuthors(nil)
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(marlow.DefaultBlueprintLimit)
		})

		g.It("allows the consumer to search for authors w/ blueprint (explicit limit)", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{Limit: 20})
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(20)
		})

		g.It("allows the consumer to search for authors w/ blueprint (explicit offset)", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{Offset: 1, Limit: 1})
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(1)
			g.Assert(authors[0].ID).Equal(2)
		})

		g.It("allows the consumer to search for authors by explicit Name", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{
				Name: []string{"author-11", "author-21", "not-exists"},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(2)
		})

		g.It("allows the consumer to search by 'NameLike'", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{
				NameLike: []string{"%-100%"},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(1)
		})

		g.It("allows the consumer to search for authors by explicit ID", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(authors)).Equal(2)
		})

		g.It("correctly serializes null/not null values into a sql.NullInt64 field", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{
				ID: []int{1337, 1338},
			})
			g.Assert(e).Equal(nil)

			g.Assert(authors[0].Name).Equal("learned author")
			g.Assert(authors[0].UniversityID.Valid).Equal(true)
			g.Assert(authors[0].UniversityID.Int64).Equal(10)

			g.Assert(authors[1].Name).Equal("other author")
			g.Assert(authors[1].UniversityID.Valid).Equal(false)
		})

		g.It("allows consumer to count w/ nil blueprint", func() {
			count, e := store.CountAuthors(nil)
			g.Assert(e).Equal(nil)
			g.Assert(count).Equal(generatedAuthorCount + 1)
		})

		g.It("allows consumer to search by authors with null UniversityID", func() {
			count, e := store.CountAuthors(&AuthorBlueprint{
				UniversityID: []sql.NullInt64{
					{Valid: false},
				},
			})
			g.Assert(e).Equal(nil)
			g.Assert(count).Equal(generatedAuthorCount)
		})

		g.It("allows consumers to select individual university ids (result being null)", func() {
			results, e := store.SelectUniversityIDs(&AuthorBlueprint{
				ID: []int{10},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(results)).Equal(1)
			g.Assert(results[0].Valid).Equal(false)
		})

		g.It("allows consumers to select individual university ids (result being valid)", func() {
			results, e := store.SelectUniversityIDs(&AuthorBlueprint{
				ID: []int{1337},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(results)).Equal(1)
			g.Assert(results[0].Valid).Equal(true)
			g.Assert(results[0].Int64).Equal(10)
		})

		g.It("allows consumers to select individual author names", func() {
			results, e := store.SelectNames(&AuthorBlueprint{
				ID: []int{10},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(results)).Equal(1)
			g.Assert(results[0]).Equal("author-101")
		})

		g.It("allows consumers to select individual author ids", func() {
			ids, e := store.SelectIDs(&AuthorBlueprint{
				ID: []int{10},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(ids)).Equal(1)
		})

		g.It("allows consumer to search by authors where not null if empty", func() {
			count, e := store.CountAuthors(&AuthorBlueprint{
				UniversityID: []sql.NullInt64{},
			})
			g.Assert(e).Equal(nil)
			g.Assert(count).Equal(1)
		})

		g.It("allows consumer to search by authors with explicit UniversityID", func() {
			count, e := store.CountAuthors(&AuthorBlueprint{
				UniversityID: []sql.NullInt64{
					{Int64: 10, Valid: true},
				},
			})
			g.Assert(e).Equal(nil)
			g.Assert(count).Equal(1)
		})

		g.It("allows the consumer to update the author id", func() {
			c, e, _ := store.UpdateAuthorID(1991, &AuthorBlueprint{ID: []int{8}})
			g.Assert(e).Equal(nil)
			g.Assert(c).Equal(1)
		})

		g.It("allows the consumer to update the author university id using nil", func() {
			c, e, _ := store.UpdateAuthorUniversityID(nil, &AuthorBlueprint{ID: []int{1}})
			g.Assert(e).Equal(nil)
			g.Assert(c).Equal(1)
		})

		g.It("allows the consumer to update the author university id using valid: false", func() {
			c, e, _ := store.UpdateAuthorUniversityID(&sql.NullInt64{Valid: false}, &AuthorBlueprint{ID: []int{2}})
			g.Assert(e).Equal(nil)
			g.Assert(c).Equal(1)
		})

		g.It("allows the consumer to update the author university id using valid: true", func() {
			c, e, _ := store.UpdateAuthorUniversityID(&sql.NullInt64{
				Valid: true,
				Int64: 101,
			}, &AuthorBlueprint{ID: []int{2}})

			g.Assert(e).Equal(nil)
			g.Assert(c).Equal(1)
		})

		g.It("allows the consumer to update the author name", func() {
			c, e := store.CountAuthors(&AuthorBlueprint{Name: []string{"danny"}})
			g.Assert(e).Equal(nil)
			g.Assert(c).Equal(0)

			updatedCount, e, q := store.UpdateAuthorName("danny", &AuthorBlueprint{ID: []int{1}})
			if e != nil {
				t.Logf("%s", q)
			}
			g.Assert(e).Equal(nil)
			g.Assert(updatedCount).Equal(1)

			a, e := store.FindAuthors(&AuthorBlueprint{ID: []int{1}})
			g.Assert(e).Equal(nil)
			g.Assert(a[0].Name).Equal("danny")
		})

		g.It("allows the consumer to count authors by blueprint", func() {
			count, e := store.CountAuthors(&AuthorBlueprint{
				ID: []int{1, 2},
			})
			g.Assert(e).Equal(nil)
			g.Assert(count).Equal(2)
		})

		g.Describe("CreateAuthors", func() {
			g.It("returns immediately with 0 if no authors", func() {
				s, e := store.CreateAuthors()
				g.Assert(e).Equal(nil)
				g.Assert(s).Equal(0)
			})

			g.It("returns the number of authors created", func() {
				s, e := store.CreateAuthors(Author{Name: "Danny"}, Author{Name: "Amelia"})
				g.Assert(e).Equal(nil)
				g.Assert(s).Equal(2)
				found, e := store.FindAuthors(&AuthorBlueprint{Name: []string{"Amelia"}})
				g.Assert(e).Equal(nil)
				g.Assert(len(found)).Equal(1)
				g.Assert(found[0].ID > 0).Equal(true)
			})
		})

		g.Describe("DeleteAuthors", func() {

			g.It("returns an error and a negative number with an empty blueprint", func() {
				c, e := store.DeleteAuthors(&AuthorBlueprint{})
				g.Assert(e == nil).Equal(false)
				g.Assert(c).Equal(-1)
			})

			g.It("returns an error and a negative number without a blueprint", func() {
				c, e := store.DeleteAuthors(nil)
				g.Assert(e == nil).Equal(false)
				g.Assert(c).Equal(-1)
			})

			g.It("successfully deletes the records found by the blueprint", func() {
				deleted, e := store.DeleteAuthors(&AuthorBlueprint{ID: []int{13}})
				g.Assert(e).Equal(nil)
				g.Assert(deleted).Equal(1)
				found, e := store.CountAuthors(&AuthorBlueprint{ID: []int{13}})
				g.Assert(e).Equal(nil)
				g.Assert(found).Equal(0)
			})

		})
	})
}
