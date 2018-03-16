package marlow

import "io"
import "fmt"
import "sync"
import "bytes"
import "testing"
import "net/url"
import "go/token"
import "go/parser"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/marlow/constants"

func Test_Blueprint(t *testing.T) {
	g := goblin.Goblin(t)

	var b *bytes.Buffer
	var r url.Values
	var f map[string]url.Values
	var record marlowRecord

	g.Describe("blueprint generator test suite", func() {

		var imports chan string
		var receivedImports map[string]bool
		var wg *sync.WaitGroup
		var closed bool

		g.BeforeEach(func() {
			imports = make(chan string, 10)
			receivedImports = make(map[string]bool)
			wg = &sync.WaitGroup{}
			closed = false

			b = new(bytes.Buffer)
			f = make(map[string]url.Values)
			r = make(url.Values)

			record = marlowRecord{
				config:        r,
				fields:        f,
				importChannel: imports,
			}

			wg.Add(1)

			go func() {
				for i := range imports {
					receivedImports[i] = true
				}

				wg.Done()
			}()

			r.Set("recordName", "Book")
		})

		g.AfterEach(func() {
			if closed == false {
				close(imports)
				wg.Wait()
			}
		})

		g.Describe("with an invalid field", func() {
			g.BeforeEach(func() {
				f["Name"] = make(url.Values)
			})

			g.It("returns an error if a field does not have a type", func() {
				reader := newBlueprintGenerator(record)
				_, e := io.Copy(b, reader)
				g.Assert(e == nil).Equal(false)
			})

			g.It("did not send any imports over the channel", func() {
				reader := newBlueprintGenerator(record)
				io.Copy(b, reader)
				close(imports)
				wg.Wait()
				g.Assert(len(receivedImports)).Equal(0)
				closed = true
			})
		})

		g.Describe("with some valid fields", func() {
			g.BeforeEach(func() {
				f["Name"] = url.Values{
					"type":   []string{"string"},
					"column": []string{"name"},
				}

				f["PageCount"] = url.Values{
					"type":   []string{"int"},
					"column": []string{"page_count"},
				}

				f["CompanyID"] = url.Values{
					"type":   []string{"sql.NullInt64"},
					"column": []string{"company_id"},
				}

				f["Birthday"] = url.Values{
					"type":   []string{"time.Time"},
					"column": []string{"birthday"},
				}

				r.Set(constants.BlueprintNameConfigOption, "SomeBlueprint")
			})

			g.It("injected the strings library to the import stream", func() {
				io.Copy(b, newBlueprintGenerator(record))
				closed = true
				close(imports)
				wg.Wait()
				g.Assert(receivedImports["strings"]).Equal(true)
			})

			g.It("produced valid a golang struct", func() {
				fmt.Fprintln(b, "package marlowt")
				_, e := io.Copy(b, newBlueprintGenerator(record))
				g.Assert(e).Equal(nil)
				_, e = parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
				g.Assert(e).Equal(nil)
			})

			g.Describe("with a postgres record dialect", func() {
				g.BeforeEach(func() {
					r.Set(constants.DialectConfigOption, "postgres")
				})

				g.It("produced valid a golang struct", func() {
					fmt.Fprintln(b, "package marlowt")
					_, e := io.Copy(b, newBlueprintGenerator(record))
					g.Assert(e).Equal(nil)
					_, e = parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
					g.Assert(e).Equal(nil)
				})
			})
		})

	})

}
