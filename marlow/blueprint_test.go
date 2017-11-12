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

		var inputs chan string
		var receivedInputs map[string]bool
		var wg *sync.WaitGroup
		var closed bool

		g.BeforeEach(func() {
			inputs = make(chan string, 10)
			receivedInputs = make(map[string]bool)
			wg = &sync.WaitGroup{}
			closed = false

			b = new(bytes.Buffer)
			f = make(map[string]url.Values)
			r = make(url.Values)

			record = marlowRecord{
				config: r,
				fields: f,
			}

			wg.Add(1)

			go func() {
				for i := range inputs {
					receivedInputs[i] = true
				}

				wg.Done()
			}()

			r.Set("recordName", "Book")
		})

		g.AfterEach(func() {
			if closed == false {
				close(inputs)
				wg.Wait()
			}
		})

		g.Describe("with an invalid field", func() {

			g.BeforeEach(func() {
				f["Name"] = make(url.Values)
			})

			g.It("returns an error if a field does not have a type", func() {
				reader := newBlueprintGenerator(record, inputs)
				_, e := io.Copy(b, reader)
				g.Assert(e == nil).Equal(false)
			})

			g.It("did not send any imports over the channel", func() {
				reader := newBlueprintGenerator(record, inputs)
				io.Copy(b, reader)
				close(inputs)
				wg.Wait()
				g.Assert(len(receivedInputs)).Equal(0)
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

				r.Set(constants.BlueprintNameConfigOption, "SomeBlueprint")
			})

			g.It("injected the strings library to the import stream", func() {
				io.Copy(b, newBlueprintGenerator(record, inputs))
				closed = true
				close(inputs)
				wg.Wait()
				g.Assert(receivedInputs["strings"]).Equal(true)
			})

			g.It("produced valid a golang struct", func() {
				fmt.Fprintln(b, "package marlowt")
				_, e := io.Copy(b, newBlueprintGenerator(record, inputs))
				g.Assert(e).Equal(nil)
				_, e = parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
				g.Assert(e).Equal(nil)
			})

		})

	})

}
