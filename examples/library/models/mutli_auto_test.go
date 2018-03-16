package models

import "os"
import "fmt"
import "testing"
import _ "github.com/lib/pq"
import "database/sql"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/examples/library/data"

func Test_MultiAuto(t *testing.T) {
	g := goblin.Goblin(t)

	var db *sql.DB
	var store MultiAutoStore

	g.Describe("Model with mutliple auto increment columns", func() {
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
			schema, e := data.Asset("data/postgres.sql")
			g.Assert(e).Equal(nil)
			_, e = db.Exec(string(schema))
			g.Assert(e).Equal(nil)
			store = NewMultiAutoStore(db, nil)
		})

		g.It("allows the consumer to create multiple records, respecting prepared index params", func() {
			_, e := store.CreateMultiAutos([]MultiAuto{
				{Name: "first"},
				{Name: "second"},
			}...)
			g.Assert(e).Equal(nil)
		})

		g.Describe("updates", func() {
			var updateBlueprint *MultiAutoBlueprint

			g.BeforeEach(func() {
				id, e := store.CreateMultiAutos(MultiAuto{Name: "updater"})
				g.Assert(e).Equal(nil)
				updateBlueprint = &MultiAutoBlueprint{ID: []uint{uint(id)}}
			})

			g.It("allows consumer to update status", func() {
				_, e := store.UpdateMultiAutoStatus("updated", updateBlueprint)
				g.Assert(e).Equal(nil)
			})

			g.It("allows consumer to update name", func() {
				_, e := store.UpdateMultiAutoName("updated", updateBlueprint)
				g.Assert(e).Equal(nil)
			})

			g.It("allows consumer to update id", func() {
				_, e := store.UpdateMultiAutoID(2000, updateBlueprint)
				g.Assert(e).Equal(nil)
			})
		})

		g.It("allows consumer to delete mutli auto records", func() {
			id, e := store.CreateMultiAutos(MultiAuto{Name: "to-delete"})
			g.Assert(e).Equal(nil)
			_, e = store.DeleteMultiAutos(&MultiAutoBlueprint{ID: []uint{uint(id)}})
			g.Assert(e).Equal(nil)
		})

		g.Describe("selections", func() {
			g.BeforeEach(func() {
				autos := make([]MultiAuto, 100)

				for i := 0; i < 100; i++ {
					autos[i] = MultiAuto{
						Name:   fmt.Sprintf("auto-%d", i),
						Status: "pending",
					}
				}

				_, e := store.CreateMultiAutos(autos...)
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to query the records (by status)", func() {
				_, e := store.FindMultiAutos(&MultiAutoBlueprint{Status: []string{"pending"}})
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to query the records (by status like)", func() {
				_, e := store.FindMultiAutos(&MultiAutoBlueprint{StatusLike: []string{"%%pending%%"}})
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to query the records (by name like)", func() {
				_, e := store.FindMultiAutos(&MultiAutoBlueprint{NameLike: []string{"%%-1%%"}})
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to query the records (by name)", func() {
				_, e := store.FindMultiAutos(&MultiAutoBlueprint{Name: []string{"first"}})
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to query the records (by id)", func() {
				_, e := store.FindMultiAutos(&MultiAutoBlueprint{ID: []uint{1}})
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to query the records (w limit)", func() {
				_, e := store.FindMultiAutos(&MultiAutoBlueprint{Limit: 1})
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to count mutli autos", func() {
				a, e := store.CountMultiAutos(&MultiAutoBlueprint{Limit: 1})
				g.Assert(e).Equal(nil)
				g.Assert(a > 0).Equal(true)
			})

			g.It("allows the consumer to select names", func() {
				_, e := store.SelectNames(&MultiAutoBlueprint{Limit: 1})
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to select statuses", func() {
				_, e := store.SelectStatuses(&MultiAutoBlueprint{Limit: 1})
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to select ids", func() {
				_, e := store.SelectIDs(&MultiAutoBlueprint{Limit: 1})
				g.Assert(e).Equal(nil)
			})
		})

	})
}
