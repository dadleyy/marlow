package models

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

		g.It("uses the postgres dialect for blueprint params", func() {
			_, e := store.CountGenres(nil)
			g.Assert(e).Equal(nil)
		})
	})
}
