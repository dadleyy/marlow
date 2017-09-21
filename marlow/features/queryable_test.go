package features

import "fmt"
import "io"
import "sync"
import "bytes"
import "net/url"
import "testing"
import "go/ast"
import "go/token"
import "go/parser"
import "github.com/franela/goblin"

type queryableTestScaffold struct {
	output   *bytes.Buffer
	imports  chan string
	record   url.Values
	fields   map[string]url.Values
	received map[string]bool
	closed   bool
	wg       *sync.WaitGroup
}

func (s *queryableTestScaffold) g() io.Reader {
	return NewQueryableGenerator(s.record, s.fields, s.imports)
}

func (s *queryableTestScaffold) parsed() (*ast.File, error) {
	return parser.ParseFile(token.NewFileSet(), "", s.output, parser.AllErrors)
}

func Test_QueryableGenerator(t *testing.T) {
	g := goblin.Goblin(t)

	var scaffold *queryableTestScaffold

	g.Describe("NewQueryableGenerator", func() {

		g.BeforeEach(func() {
			scaffold = &queryableTestScaffold{
				output:   new(bytes.Buffer),
				imports:  make(chan string),
				record:   make(url.Values),
				fields:   make(map[string]url.Values),
				received: make(map[string]bool),
				closed:   false,
				wg:       &sync.WaitGroup{},
			}

			scaffold.wg.Add(1)

			go func() {
				for i := range scaffold.imports {
					scaffold.received[i] = true
				}
				scaffold.wg.Done()
			}()
		})

		g.AfterEach(func() {
			if scaffold.closed == false {
				close(scaffold.imports)
				scaffold.wg.Wait()
			}
		})

		g.Describe("with invalid record config", func() {
			g.AfterEach(func() {
				g.Assert(scaffold.received["fmt"]).Equal(false)
				g.Assert(scaffold.received["strings"]).Equal(false)
				g.Assert(scaffold.received["bytes"]).Equal(false)
			})

			g.It("returns an error if the record does not have a table, recordName or storeName", func() {
				_, e := io.Copy(scaffold.output, scaffold.g())
				g.Assert(e == nil).Equal(false)
			})

			g.It("returns an error if the record does not have a table or storeName but a valid recordName", func() {
				scaffold.record.Set("recordName", "Book")
				_, e := io.Copy(scaffold.output, scaffold.g())
				g.Assert(e == nil).Equal(false)
			})

			g.It("returns an error if the record does not have a table or recordName but a valid storeName", func() {
				scaffold.record.Set("storeName", "BookStore")
				_, e := io.Copy(scaffold.output, scaffold.g())
				g.Assert(e == nil).Equal(false)
			})

			g.It("returns an error if the record does not have a table or recordName but a valid table", func() {
				scaffold.record.Set("table", "books")
				_, e := io.Copy(scaffold.output, scaffold.g())
				g.Assert(e == nil).Equal(false)
			})

			g.It("acts as a no-op for valid records with zero fields", func() {
				scaffold.record.Set("storeName", "BookStore")
				scaffold.record.Set("recordName", "Book")
				scaffold.record.Set("table", "books")
				_, e := io.Copy(scaffold.output, scaffold.g())
				g.Assert(e).Equal(nil)
				g.Assert(scaffold.output.Len()).Equal(0)
				scaffold.closed = true
				close(scaffold.imports)
				scaffold.wg.Wait()
				g.Assert(len(scaffold.received)).Equal(0)
			})
		})

		g.Describe("with a valid record configuration and some fields", func() {

			g.BeforeEach(func() {
				scaffold.record.Set("storeName", "BookStore")
				scaffold.record.Set("recordName", "Book")
				scaffold.record.Set("table", "books")
				scaffold.fields["Title"] = url.Values{
					"type":   []string{"string"},
					"column": []string{"title"},
				}
			})

			g.It("produces valid golang code", func() {
				fmt.Fprintln(scaffold.output, "package marlowt")
				io.Copy(scaffold.output, scaffold.g())
				tree, e := scaffold.parsed()
				g.Assert(e).Equal(nil)
				g.Assert(tree == nil).Equal(false)
			})

			g.It("injected the fmt, bytes and strings packages to the import channel", func() {
				fmt.Fprintln(scaffold.output, "package marlowt")
				io.Copy(scaffold.output, scaffold.g())
				scaffold.closed = true
				close(scaffold.imports)
				scaffold.wg.Wait()

				g.Assert(scaffold.received["fmt"]).Equal(true)
				g.Assert(scaffold.received["strings"]).Equal(true)
				g.Assert(scaffold.received["bytes"]).Equal(true)
			})
		})

	})
}
