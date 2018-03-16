package models

import "os"
import "io"
import "fmt"
import "time"
import "bytes"
import "strings"
import "testing"
import _ "github.com/mattn/go-sqlite3"
import "database/sql"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/marlow"

func addAuthorRow(db *sql.DB, values ...[]string) error {
	for _, rowValues := range values {
		valueString := strings.Join(rowValues, ",")
		statement := fmt.Sprintf("insert into authors (system_id,name,birthday) values(%s);", valueString)
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
	var queryLog io.Writer

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

			g.Assert(r).Equal("WHERE (authors.system_id > ? AND authors.system_id < ?)")
		})

		g.It("supports in on uint column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{AuthorFlags: []uint8{1, 2}})
			g.Assert(r).Equal("WHERE authors.flags IN (?,?)")
		})

		g.It("supports range on uint column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{AuthorFlagsRange: []uint8{1, 2}})
			g.Assert(r).Equal("WHERE (authors.flags > ? AND authors.flags < ?)")
		})

		g.It("supports range on float64 column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{ReaderRatingRange: []float64{1, 2}})
			g.Assert(r).Equal("WHERE (authors.rating > ? AND authors.rating < ?)")
		})

		g.It("supports 'IN' on float64 column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{ReaderRating: []float64{1, 2, 3}})
			g.Assert(r).Equal("WHERE authors.rating IN (?,?,?)")
		})

		g.It("supports 'IN' on ID column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{ID: []int{1, 2, 3}})
			g.Assert(r).Equal("WHERE authors.system_id IN (?,?,?)")
		})

		g.It("supports a combination of range and 'IN' on ID column querying", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{
				ID:      []int{1, 2, 3},
				IDRange: []int{1, 4},
			})

			g.Assert(r).Equal("WHERE authors.system_id IN (?,?,?) AND (authors.system_id > ? AND authors.system_id < ?)")
		})

		g.It("supports making a blueprint inclusive", func() {
			r := fmt.Sprintf("%s", &AuthorBlueprint{
				NameLike:  []string{"%rodger%"},
				IDRange:   []int{1, 4},
				Inclusive: true,
			})

			g.Assert(r).Equal("WHERE (authors.system_id > ? AND authors.system_id < ?) OR authors.name LIKE ?")
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
				authors = append(authors, []string{id, name, "date()"})
			}

			g.Assert(addAuthorRow(db, authors...)).Equal(nil)

			_, e = db.Exec(
				"insert into authors (system_id,name,university_id,birthday) values(1337,'learned author',10,date());",
			)
			g.Assert(e).Equal(nil)

			_, e = db.Exec(
				"insert into authors (system_id,name,university_id,birthday) values(1338,'other author',null,date());",
			)
			g.Assert(e).Equal(nil)
		})

		g.BeforeEach(func() {
			queryLog = new(bytes.Buffer)
			store = NewAuthorStore(db, queryLog)
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

		g.It("allows the consumer to search for authors by ID range", func() {
			authors, e := store.FindAuthors(&AuthorBlueprint{
				IDRange: []int{1, 4},
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

		g.It("allows consumers to select individual university ids with limit", func() {
			var uniID sql.NullInt64
			uniID.Scan(10)
			_, e := store.CreateAuthors([]Author{
				{Name: "limited author 001"},
				{Name: "limited author 002", UniversityID: uniID},
				{Name: "limited author 003"},
			}...)
			results, e := store.SelectAuthorUniversityIDs(&AuthorBlueprint{
				NameLike: []string{"%limited%"},
				Limit:    1,
				Offset:   1,
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(results)).Equal(1)
			g.Assert(results[0].Int64).Equal(10)
			store.DeleteAuthors(&AuthorBlueprint{
				NameLike: []string{"%limited%"},
			})
		})

		g.It("allows consumers to select individual university ids (result being null)", func() {
			results, e := store.SelectAuthorUniversityIDs(&AuthorBlueprint{
				ID: []int{10},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(results)).Equal(1)
			g.Assert(results[0].Valid).Equal(false)
		})

		g.It("allows consumers to select individual university ids (result being valid)", func() {
			results, e := store.SelectAuthorUniversityIDs(&AuthorBlueprint{
				ID: []int{1337},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(results)).Equal(1)
			g.Assert(results[0].Valid).Equal(true)
			g.Assert(results[0].Int64).Equal(10)
		})

		g.It("allows consumers to select individual author names with limit", func() {
			_, e := store.CreateAuthors([]Author{
				{Name: "limited name 001"},
				{Name: "limited name 002"},
				{Name: "limited name 003"},
				{Name: "limited name 004"},
			}...)
			results, e := store.SelectAuthorNames(&AuthorBlueprint{
				NameLike: []string{"%limited name%"},
				Limit:    1,
				Offset:   1,
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(results)).Equal(1)
			g.Assert(results[0]).Equal("limited name 002")
			store.DeleteAuthors(&AuthorBlueprint{
				NameLike: []string{"%limited name%"},
			})
		})

		g.It("allows consumers to select individual author names", func() {
			results, e := store.SelectAuthorNames(&AuthorBlueprint{
				ID: []int{10},
			})
			g.Assert(e).Equal(nil)
			g.Assert(len(results)).Equal(1)
			g.Assert(results[0]).Equal("author-101")
		})

		g.It("allows consumers to select individual author ids", func() {
			ids, e := store.SelectAuthorIDs(&AuthorBlueprint{
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
			c, e := store.UpdateAuthorID(1991, &AuthorBlueprint{ID: []int{8}})
			g.Assert(e).Equal(nil)
			g.Assert(c).Equal(1)
		})

		g.It("allows the consumer to update the author university id using nil", func() {
			c, e := store.UpdateAuthorUniversityID(nil, &AuthorBlueprint{ID: []int{1}})
			g.Assert(e).Equal(nil)
			g.Assert(c).Equal(1)
		})

		g.It("allows the consumer to update the author university id using valid: false", func() {
			c, e := store.UpdateAuthorUniversityID(&sql.NullInt64{Valid: false}, &AuthorBlueprint{ID: []int{2}})
			g.Assert(e).Equal(nil)
			g.Assert(c).Equal(1)
		})

		g.It("allows the consumer to update the author university id using valid: true", func() {
			c, e := store.UpdateAuthorUniversityID(&sql.NullInt64{
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

			updatedCount, e := store.UpdateAuthorName("danny", &AuthorBlueprint{ID: []int{1}})
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

			g.It("returns the id of the latest author created", func() {
				s, e := store.CreateAuthors([]Author{
					{Name: "Danny", ReaderRating: 25.00},
					{Name: "Amelia", ReaderRating: 99.99},
				}...)
				g.Assert(e).Equal(nil)

				found, e := store.FindAuthors(&AuthorBlueprint{ID: []int{int(s)}})
				g.Assert(e).Equal(nil)
				g.Assert(len(found)).Equal(1)
				g.Assert(found[0].Name).Equal("Amelia")

				badReaderCount, e := store.CountAuthors(&AuthorBlueprint{
					ReaderRating: []float64{25.00},
				})
				g.Assert(e).Equal(nil)
				g.Assert(badReaderCount).Equal(1)

				goodReaderCount, e := store.CountAuthors(&AuthorBlueprint{
					ReaderRating: []float64{99.99},
				})
				g.Assert(e).Equal(nil)
				g.Assert(goodReaderCount).Equal(1)
			})
		})

		g.Describe("uint8 field interactions", func() {

			g.It("allows the consumer to create records w/ uint8 fields", func() {
				_, e := store.CreateAuthors(Author{
					Name:        "flaot64 testing",
					AuthorFlags: 3,
				})
				g.Assert(e).Equal(nil)

				_, e = store.DeleteAuthors(&AuthorBlueprint{
					AuthorFlagsRange: []uint8{1, 4},
				})
				g.Assert(e).Equal(nil)
			})

			g.It("allows users to select uint8 fields", func() {
				ratings, e := store.SelectAuthorAuthorFlags(&AuthorBlueprint{ID: []int{20}})
				g.Assert(e).Equal(nil)
				g.Assert(ratings[0]).Equal(uint8(0))
			})

			g.It("allows users to update uint8 fields", func() {
				blueprint := &AuthorBlueprint{ID: []int{20}}
				_, e := store.UpdateAuthorAuthorFlags(5, blueprint)
				g.Assert(e).Equal(nil)
				authors, e := store.FindAuthors(blueprint)
				g.Assert(e).Equal(nil)
				g.Assert(authors[0].AuthorFlags).Equal(uint8(5))
			})

		})

		g.Describe("time.Time field interactions", func() {
			fyodor := "fyodor dostoevsky"

			g.Before(func() {
				var uni sql.NullInt64
				uni.Scan(100)

				birthday, e := time.Parse(time.RFC3339, "1821-11-11T15:04:05Z")
				g.Assert(e).Equal(nil)

				store.CreateAuthors(Author{
					Name:         fyodor,
					UniversityID: uni,
					ReaderRating: 100.00,
					Birthday:     birthday,
				})
			})

			g.It("allows users to select birthdays", func() {
				b, e := store.SelectAuthorBirthdays(&AuthorBlueprint{Name: []string{fyodor}})
				g.Assert(e).Equal(nil)
				g.Assert(len(b)).Equal(1)
				g.Assert(b[0].Year()).Equal(1821)
			})

			g.It("allows users to update birthdays", func() {
				birthday, e := time.Parse(time.RFC3339, "1991-05-26T00:00:00Z")
				g.Assert(e).Equal(nil)
				_, e = store.UpdateAuthorBirthday(birthday, &AuthorBlueprint{
					ID: []int{1},
				})
				g.Assert(e).Equal(nil)
				a, e := store.SelectAuthorBirthdays(&AuthorBlueprint{ID: []int{1}})
				g.Assert(e).Equal(nil)
				g.Assert(len(a)).Equal(1)
				g.Assert(a[0].Year()).Equal(1991)
			})

			g.It("allows users to search by birthday range", func() {
				start, e := time.Parse(time.RFC3339, "1821-01-01T15:04:05Z")
				g.Assert(e).Equal(nil)
				end, e := time.Parse(time.RFC3339, "1850-01-01T15:04:05Z")
				a, e := store.CountAuthors(&AuthorBlueprint{BirthdayRange: []time.Time{start, end}})
				g.Assert(e).Equal(nil)
				g.Assert(a).Equal(1)
			})

			g.It("allows users to search by birthday", func() {
				birthday, e := time.Parse(time.RFC3339, "1821-11-11T15:04:05Z")
				g.Assert(e).Equal(nil)
				b, e := store.FindAuthors(&AuthorBlueprint{Birthday: []time.Time{birthday}})
				g.Assert(e).Equal(nil)
				g.Assert(len(b)).Equal(1)
			})
		})

		g.Describe("float64 field interactions", func() {

			g.It("allows the consumer to create records w/ float64 fields", func() {
				_, e := store.CreateAuthors(Author{
					Name:         "flaot64 testing",
					ReaderRating: 89.99,
				})
				g.Assert(e).Equal(nil)

				_, e = store.DeleteAuthors(&AuthorBlueprint{
					ReaderRatingRange: []float64{89, 90},
				})
				g.Assert(e).Equal(nil)
			})

			g.It("allows users to select float64 fields", func() {
				ratings, e := store.SelectAuthorReaderRatings(&AuthorBlueprint{ID: []int{20}})
				g.Assert(e).Equal(nil)
				g.Assert(ratings[0]).Equal(100.00)
			})

			g.It("allows users to update float64 fields", func() {
				blueprint := &AuthorBlueprint{ID: []int{20}}
				_, e := store.UpdateAuthorReaderRating(50.00, blueprint)
				g.Assert(e).Equal(nil)
				authors, e := store.FindAuthors(blueprint)
				g.Assert(e).Equal(nil)
				g.Assert(authors[0].ReaderRating).Equal(50.00)
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
