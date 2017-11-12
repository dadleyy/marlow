package marlow

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

import "github.com/dadleyy/marlow/marlow/writing"

type queryableTestScaffold struct {
	output *bytes.Buffer

	imports chan string
	methods chan writing.FuncDecl
	record  url.Values
	fields  map[string]url.Values

	received map[string]bool
	closed   bool
	wg       *sync.WaitGroup
}

func (s *queryableTestScaffold) g() io.Reader {
	record := marlowRecord{
		config:        s.record,
		fields:        s.fields,
		importChannel: s.imports,
		storeChannel:  s.methods,
	}
	return newQueryableGenerator(record)
}

func (s *queryableTestScaffold) close() {
	if s == nil || s.closed {
		return
	}

	s.closed = true
	close(s.methods)
	close(s.imports)
	s.wg.Wait()
}

func (s *queryableTestScaffold) parsed() (*ast.File, error) {
	return parser.ParseFile(token.NewFileSet(), "", s.output, parser.AllErrors)
}

func Test_QueryableGenerator(t *testing.T) {
	g := goblin.Goblin(t)

	var scaffold *queryableTestScaffold

	g.Describe("queryable feature test suite", func() {

		g.BeforeEach(func() {
			scaffold = &queryableTestScaffold{
				output:   new(bytes.Buffer),
				imports:  make(chan string),
				methods:  make(chan writing.FuncDecl),
				record:   make(url.Values),
				fields:   make(map[string]url.Values),
				received: make(map[string]bool),
				closed:   false,
				wg:       &sync.WaitGroup{},
			}

			scaffold.wg.Add(2)

			go func() {
				for range scaffold.methods {
				}
				scaffold.wg.Done()
			}()

			go func() {
				for i := range scaffold.imports {
					scaffold.received[i] = true
				}
				scaffold.wg.Done()
			}()
		})

		g.AfterEach(func() {
			scaffold.close()
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
				scaffold.record.Set("tableName", "books")
				_, e := io.Copy(scaffold.output, scaffold.g())
				g.Assert(e == nil).Equal(false)
			})

			g.It("acts as a no-op for valid records with zero fields", func() {
				scaffold.record.Set("defaultLimit", "100")
				scaffold.record.Set("storeName", "BookStore")
				scaffold.record.Set("recordName", "Book")
				scaffold.record.Set("tableName", "books")

				_, e := io.Copy(scaffold.output, scaffold.g())
				g.Assert(e).Equal(nil)
				g.Assert(scaffold.output.Len()).Equal(0)
				scaffold.close()
				g.Assert(len(scaffold.received)).Equal(0)
			})
		})

		g.Describe("with a valid record configuration and some fields", func() {

			g.BeforeEach(func() {
				scaffold.record.Set("defaultLimit", "20")
				scaffold.record.Set("storeName", "BookStore")
				scaffold.record.Set("blueprintName", "BookBlueprint")
				scaffold.record.Set("recordName", "Book")
				scaffold.record.Set("tableName", "books")
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
				scaffold.close()

				g.Assert(scaffold.received["fmt"]).Equal(true)
				g.Assert(scaffold.received["strings"]).Equal(true)
				g.Assert(scaffold.received["bytes"]).Equal(true)
			})
		})

	})
}
