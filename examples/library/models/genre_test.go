package models

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

		g.BeforeEach(func() {
			var e error
			db, e = sql.Open("postgres", "user=postgres dbname=marlow_test sslmode=disable")
			g.Assert(e).Equal(nil)
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

		g.It("allows the consumer to select genres by name like", func() {
			_, e := store.CountGenres(&GenreBlueprint{
				Name:    []string{"horror"},
				IDRange: []uint{0, 10},
			})
			g.Assert(e).Equal(nil)
		})
	})
}
