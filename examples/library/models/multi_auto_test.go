package models

import "os"
import "fmt"
import "time"
import "testing"
import "bytes"
import "github.com/lib/pq"
import "database/sql"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/examples/library/data"

func Test_MultiAuto(t *testing.T) {
	g := goblin.Goblin(t)

	var db *sql.DB
	var store MultiAutoStore
	var output *bytes.Buffer

	g.Describe("Model with multiple auto increment columns", func() {
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
			output = new(bytes.Buffer)
			schema, e := data.Asset("data/postgres.sql")
			g.Assert(e).Equal(nil)
			_, e = db.Exec(string(schema))
			g.Assert(e).Equal(nil)
			store = NewMultiAutoStore(db, output)
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

		g.It("allows consumer to delete multi auto records", func() {
			id, e := store.CreateMultiAutos(MultiAuto{Name: "to-delete"})
			g.Assert(e).Equal(nil)
			_, e = store.DeleteMultiAutos(&MultiAutoBlueprint{ID: []uint{uint(id)}})
			g.Assert(e).Equal(nil)
		})

		g.Describe("timestamp operations", func() {
			var blueprint MultiAutoBlueprint
			var created time.Time

			g.BeforeEach(func() {
				created = time.Now()
				id, e := store.CreateMultiAutos(MultiAuto{
					Name:      "to-update-timestamps",
					CreatedAt: created,
				})
				g.Assert(e).Equal(nil)
				blueprint = MultiAutoBlueprint{ID: []uint{uint(id)}}
			})

			g.AfterEach(func() {
				_, e := store.DeleteMultiAutos(&blueprint)
				g.Assert(e).Equal(nil)
			})

			g.It("did not insert a value for the deletedAt column", func() {
				m, e := store.SelectMultiAutoDeletedAts(&blueprint)
				g.Assert(e).Equal(nil)
				g.Assert(m[0].Valid).Equal(false)
			})

			g.It("returns the set of records not deleted by default", func() {
				c, e := store.CountMultiAutos(&blueprint)
				g.Assert(e).Equal(nil)
				g.Assert(c).Equal(1)

				_, e = store.DeleteMultiAutos(&blueprint)
				g.Assert(e).Equal(nil)

				c, e = store.CountMultiAutos(&blueprint)
				g.Assert(e).Equal(nil)
				g.Assert(c).Equal(0)

				c, e = store.CountMultiAutos(&MultiAutoBlueprint{
					ID:       blueprint.ID,
					Unscoped: true,
				})
				g.Assert(e).Equal(nil)
				g.Assert(c).Equal(1)
			})

			g.It("allows the user to search by deleted_at timestamps (range)", func() {
				start := pq.NullTime{}
				end := pq.NullTime{}
				start.Scan(created)
				end.Scan(created.Add(time.Since(created)))
				count, e := store.SelectMultiAutoDeletedAts(&MultiAutoBlueprint{DeletedAtRange: []pq.NullTime{start, end}})
				g.Assert(e).Equal(nil)
				g.Assert(len(count)).Equal(0)
			})

			g.It("allows the user to search by deleted_at timestamps (exact)", func() {
				exact := pq.NullTime{}
				exact.Scan(created)
				count, e := store.SelectMultiAutoDeletedAts(&MultiAutoBlueprint{DeletedAt: []pq.NullTime{exact}})
				g.Assert(e).Equal(nil)
				g.Assert(len(count)).Equal(0)
			})

			g.It("allows the user to search by created_at timestamps (range)", func() {
				start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
				end := time.Now().Add(time.Since(start))
				count, e := store.SelectMultiAutoDeletedAts(&MultiAutoBlueprint{
					CreatedAtRange: []time.Time{start, end},
				})
				g.Assert(e).Equal(nil)
				g.Assert(len(count)).Equal(1)
			})

			g.It("allows the user to search by created_at timestamps (exact)", func() {
				count, e := store.SelectMultiAutoDeletedAts(&MultiAutoBlueprint{CreatedAt: []time.Time{created}})
				g.Assert(e).Equal(nil)
				g.Assert(len(count)).Equal(1)
			})

			g.It("allows the user to search by created_at timestamps", func() {
				start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
				end := time.Now().Add(time.Since(start))
				count, e := store.SelectMultiAutoDeletedAts(&MultiAutoBlueprint{
					CreatedAtRange: []time.Time{start, end},
				})
				g.Assert(e).Equal(nil)
				g.Assert(len(count)).Equal(1)
			})

			g.It("allows the user to select destroyed_at timestamps", func() {
				_, e := store.SelectMultiAutoDeletedAts(&blueprint)
				g.Assert(e).Equal(nil)
			})

			g.It("allows the user to select created_at timestamps", func() {
				_, e := store.SelectMultiAutoCreatedAts(&blueprint)
				g.Assert(e).Equal(nil)
			})

			g.It("is allowed to update the created_at timestamp", func() {
				_, e := store.UpdateMultiAutoCreatedAt(time.Now(), &blueprint)
				g.Assert(e).Equal(nil)
			})

			g.It("is allowed to update the created_at timestamp", func() {
				now := pq.NullTime{}
				now.Scan(time.Now())
				_, e := store.UpdateMultiAutoDeletedAt(now, &blueprint)
				g.Assert(e).Equal(nil)
			})
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

			g.It("allows the consumer to count multi autos", func() {
				a, e := store.CountMultiAutos(&MultiAutoBlueprint{Limit: 1})
				g.Assert(e).Equal(nil)
				g.Assert(a > 0).Equal(true)
			})

			g.It("allows the consumer to select names", func() {
				_, e := store.SelectMultiAutoNames(&MultiAutoBlueprint{Limit: 1})
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to select statuses", func() {
				_, e := store.SelectMultiAutoStatuses(&MultiAutoBlueprint{Limit: 1})
				g.Assert(e).Equal(nil)
			})

			g.It("allows the consumer to select ids", func() {
				_, e := store.SelectMultiAutoIDs(&MultiAutoBlueprint{Limit: 1})
				g.Assert(e).Equal(nil)
			})
		})

	})
}
