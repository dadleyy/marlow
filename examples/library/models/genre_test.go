package models

import "os"
import "fmt"
import "testing"
import _ "github.com/lib/pq"
import "database/sql"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/examples/library/data"

func Test_Genre(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("genre record test suite (postgres)", func() {
		var db *sql.DB
		var store GenreStore

		g.Before(func() {
			var e error
			config := struct {
				username string
				database string
				port     string
			}{"postgres", "marlow_test", "5432"}

			if port := os.Getenv("PG_PORT"); len(port) > 0 {
				config.port = port
			}

			constr := fmt.Sprintf("user=%s dbname=%s port=%s sslmode=disable", config.username, config.database, config.port)
			db, e = sql.Open("postgres", constr)
			g.Assert(e).Equal(nil)
		})

		g.BeforeEach(func() {
			schema, e := data.Asset("data/genres.sql")
			g.Assert(e).Equal(nil)
			_, e = db.Exec(string(schema))
			g.Assert(e).Equal(nil)
			store = NewGenreStore(db)
		})

		g.It("uses the postgres dialect for blueprint string in params", func() {
			s := fmt.Sprintf("%s", &GenreBlueprint{Name: []string{"horror", "comedy"}})
			g.Assert(s).Equal("WHERE genres.name IN ($1,$2)")
		})

		g.It("uses the postgres dialect for blueprint int in params", func() {
			s := fmt.Sprintf("%s", &GenreBlueprint{ID: []uint{0, 10}})
			g.Assert(s).Equal("WHERE genres.id IN ($1,$2)")
		})

		g.It("uses the postgres dialect for blueprint int range params", func() {
			s := fmt.Sprintf("%s", &GenreBlueprint{IDRange: []uint{0, 10}})
			g.Assert(s).Equal("WHERE genres.id > $1 AND genres.id < $2")
		})

		g.It("uses the postgres dialect for blueprint string like params", func() {
			s := fmt.Sprintf("%s", &GenreBlueprint{NameLike: []string{"danny"}})
			g.Assert(s).Equal("WHERE genres.name LIKE $1")
		})

		g.It("allows user to create genres", func() {
			id, e := store.CreateGenres([]Genre{
				{Name: "Comedy"},
				{Name: "Literature"},
				{Name: "Science Fiction", ParentID: sql.NullInt64{Valid: true, Int64: 10}},
			}...)
			g.Assert(e).Equal(nil)

			results, e := store.SelectNames(&GenreBlueprint{
				ID: []uint{uint(id)},
			})

			g.Assert(e).Equal(nil)
			g.Assert(len(results)).Equal(1)
			g.Assert(results[0]).Equal("Science Fiction")
		})

		g.Describe("having created some genres", func() {
			var lastID int64

			g.BeforeEach(func() {
				var e error
				lastID, e = store.CreateGenres([]Genre{
					{Name: "Romance"},
					{Name: "Comedy"},
					{Name: "Literature"},
					{Name: "Science Fiction", ParentID: sql.NullInt64{Valid: true, Int64: 10}},
				}...)
				g.Assert(e).Equal(nil)
			})

			g.It("supports selecting parent ids", func() {
				parents, e := store.SelectParentIDs(&GenreBlueprint{
					ID: []uint{uint(lastID)},
				})
				g.Assert(e).Equal(nil)
				g.Assert(len(parents)).Equal(1)
				g.Assert(parents[0].Valid).Equal(true)
				g.Assert(parents[0].Int64).Equal(10)
			})

			g.It("allows counting w/ empty NullInt64 blueprint (NOT NULL)", func() {
				c, e := store.CountGenres(&GenreBlueprint{
					ParentID: []sql.NullInt64{},
				})
				g.Assert(e).Equal(nil)
				g.Assert(c).Equal(1)
			})

			g.It("allows counting w/ valid NullInt64 blueprint", func() {
				var p sql.NullInt64
				p.Scan(10)
				children, e := store.CountGenres(&GenreBlueprint{
					ParentID: []sql.NullInt64{p},
				})
				g.Assert(e).Equal(nil)
				g.Assert(children).Equal(1)
			})

			g.It("allows counting w/ nil NullInt64 blueprint", func() {
				var p sql.NullInt64
				p.Scan(nil)
				orphans, e := store.CountGenres(&GenreBlueprint{
					ParentID: []sql.NullInt64{p},
				})
				g.Assert(e).Equal(nil)
				g.Assert(orphans).Equal(3)
			})

			g.It("allows selecting by genre name like", func() {
				ids, e := store.SelectIDs(&GenreBlueprint{
					NameLike: []string{"%%Fiction%%"},
				})
				g.Assert(e).Equal(nil)
				g.Assert(len(ids)).Equal(1)
			})

			g.It("allows finding by id range (with offset and limit)", func() {
				genres, e := store.FindGenres(&GenreBlueprint{
					IDRange: []uint{0, 10},
					Offset:  1,
					Limit:   1,
				})
				g.Assert(e).Equal(nil)
				g.Assert(len(genres)).Equal(1)
				g.Assert(genres[0].ID).Equal(uint(2))
			})

			g.It("allows finding by id range", func() {
				genres, e := store.FindGenres(&GenreBlueprint{
					IDRange: []uint{0, 10},
				})
				g.Assert(e).Equal(nil)
				g.Assert(len(genres)).Equal(4)
			})

			g.It("allows updating the genre name", func() {
				bp := &GenreBlueprint{ID: []uint{1}}
				_, e, _ := store.UpdateGenreName("Politics", bp)
				g.Assert(e).Equal(nil)

				names, e := store.SelectNames(bp)
				g.Assert(e).Equal(nil)
				g.Assert(len(names)).Equal(1)
				g.Assert(names[0]).Equal("Politics")
			})

			g.It("allows deleting a newly created genre", func() {
				id, e := store.CreateGenres(Genre{
					Name: "Comic Books",
				})
				g.Assert(e).Equal(nil)

				_, e = store.DeleteGenres(&GenreBlueprint{
					ID: []uint{uint(id)},
				})
				g.Assert(e).Equal(nil)

				c, e := store.CountGenres(&GenreBlueprint{
					Name: []string{"Comic Books"},
				})
				g.Assert(e).Equal(nil)
				g.Assert(c).Equal(0)
			})

			g.It("allows updating the genre parent id", func() {
				var p sql.NullInt64
				p.Scan(100)

				bp := &GenreBlueprint{ID: []uint{1}}
				_, e, _ := store.UpdateGenreParentID(&p, bp)
				g.Assert(e).Equal(nil)

				ids, e := store.SelectParentIDs(bp)

				g.Assert(e).Equal(nil)
				g.Assert(len(ids)).Equal(1)
				g.Assert(ids[0].Valid).Equal(true)
				g.Assert(ids[0].Int64).Equal(100)

				p.Scan(nil)
				_, e, _ = store.UpdateGenreParentID(&p, bp)
				g.Assert(e).Equal(nil)

				ids, e = store.SelectParentIDs(bp)
				g.Assert(e).Equal(nil)
				g.Assert(len(ids)).Equal(1)
				g.Assert(ids[0].Valid).Equal(false)
			})

			g.It("allows updating the genre id", func() {
				bp := &GenreBlueprint{Name: []string{"Science Fiction"}}
				_, e, _ := store.UpdateGenreID(1337, bp)
				g.Assert(e).Equal(nil)

				ids, e := store.SelectIDs(bp)
				g.Assert(e).Equal(nil)
				g.Assert(len(ids)).Equal(1)
				g.Assert(ids[0]).Equal(uint(1337))
			})
		})

		g.It("allows the consumer to select genres by name like", func() {
			_, e := store.CountGenres(&GenreBlueprint{
				Name:    []string{"horror"},
				IDRange: []uint{0, 10},
			})
			g.Assert(e).Equal(nil)
		})
	})
}
