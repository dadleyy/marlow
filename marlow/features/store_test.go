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

type storeTestScaffold struct {
	ouput    *bytes.Buffer
	imports  chan string
	record   url.Values
	received map[string]bool
	closed   bool
	wg       *sync.WaitGroup
}

func (s *storeTestScaffold) g() io.Reader {
	return NewStoreGenerator(s.record, s.imports)
}

func (s *storeTestScaffold) parsed() (*ast.File, error) {
	return parser.ParseFile(token.NewFileSet(), "", s.ouput, parser.AllErrors)
}

func (s *storeTestScaffold) close() {
	s.closed = true
	close(s.imports)
	s.wg.Wait()
}

func Test_StoreGenerator(t *testing.T) {
	g := goblin.Goblin(t)

	var scaffold *storeTestScaffold

	g.Describe("NewStoreGenerator", func() {

		g.BeforeEach(func() {
			scaffold = &storeTestScaffold{
				ouput:    new(bytes.Buffer),
				imports:  make(chan string),
				record:   make(url.Values),
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
				scaffold.close()
			}
		})

		g.It("returns an error if the store name is not valid", func() {
			_, e := io.Copy(scaffold.ouput, scaffold.g())
			g.Assert(e == nil).Equal(false)
		})

		g.It("does not inject any imports if name is invalid", func() {
			io.Copy(scaffold.ouput, scaffold.g())
			scaffold.close()
			g.Assert(len(scaffold.received)).Equal(0)
		})

		g.Describe("with a valid store name", func() {

			g.BeforeEach(func() {
				scaffold.record.Set("storeName", "BookStore")
				fmt.Fprintln(scaffold.ouput, "package marlowt")
			})

			g.It("injects fmt and sql packages into import stream", func() {
				io.Copy(scaffold.ouput, scaffold.g())
				scaffold.close()
				g.Assert(scaffold.received["fmt"]).Equal(true)
				g.Assert(scaffold.received["database/sql"]).Equal(true)
			})

			g.It("writes valid golang code if store name is present", func() {
				io.Copy(scaffold.ouput, scaffold.g())
				_, e := scaffold.parsed()
				g.Assert(e).Equal(nil)
			})
		})
	})
}
