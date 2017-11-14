package marlow

import "io"
import "sync"
import "bytes"
import "go/ast"
import "testing"
import "strings"
import "go/token"
import "go/parser"
import "github.com/franela/goblin"

type recordReaderTestScaffold struct {
	source  io.Reader
	imports chan string
	output  *bytes.Buffer
	waiter  *sync.WaitGroup
	closed  bool
}

func (s *recordReaderTestScaffold) root() ast.Decl {
	tree, e := parser.ParseFile(token.NewFileSet(), "", s.source, parser.AllErrors)

	if e != nil {
		panic(e)
	}

	if len(tree.Decls) < 1 {
		panic("not enough declarations in provided source")
	}

	return tree.Decls[0]
}

func (s *recordReaderTestScaffold) close() {
	if s.closed {
		return
	}

	s.closed = true
	close(s.imports)
}

func (s *recordReaderTestScaffold) error() error {
	reader, _ := newRecordReader(s.root(), s.imports)
	_, e := io.Copy(s.output, reader)
	return e
}

func (s *recordReaderTestScaffold) reset() {
	s.closed = false
	s.imports = make(chan string)
	s.output = new(bytes.Buffer)
	s.waiter = &sync.WaitGroup{}
}

func Test_RecordReader(t *testing.T) {
	g := goblin.Goblin(t)

	var scaffold *recordReaderTestScaffold

	g.Describe("recordReader", func() {

		g.BeforeEach(func() {
			scaffold = &recordReaderTestScaffold{}
			scaffold.reset()
		})

		g.BeforeEach(func() {
			scaffold.waiter.Add(1)

			go func() {
				received := make(map[string]bool)

				for importName := range scaffold.imports {
					received[importName] = true
				}

				scaffold.waiter.Done()
			}()
		})

		g.AfterEach(func() {
			scaffold.close()
			scaffold.waiter.Wait()
		})

		g.It("with a valid source struct", func() {
			scaffold.source = strings.NewReader(`
				package marlowt
				type Author struct {
					Title string
				}`)
			reader, ok := newRecordReader(scaffold.root(), scaffold.imports)
			g.Assert(ok).Equal(true)
			_, e := io.Copy(scaffold.output, reader)
			scaffold.close()
			g.Assert(e).Equal(nil)
		})

		g.It("with a valid source struct including explicit column names", func() {
			scaffold.source = strings.NewReader(`
					package marlowt
					type Author struct {
						Title string ` + "`marlow:\"column=title\"`" + `
					}`)
			reader, ok := newRecordReader(scaffold.root(), scaffold.imports)
			g.Assert(ok).Equal(true)
			_, e := io.Copy(scaffold.output, reader)
			scaffold.close()
			g.Assert(e).Equal(nil)
		})

		g.It("with a valid source struct including explicit table name from field", func() {
			scaffold.source = strings.NewReader(`
					package marlowt
					type Author struct {
						table string ` + "`marlow:\"tableName=authors\"`" + `
						Title string
					}`)
			reader, ok := newRecordReader(scaffold.root(), scaffold.imports)
			g.Assert(ok).Equal(true)
			_, e := io.Copy(scaffold.output, reader)
			scaffold.close()
			g.Assert(e).Equal(nil)
		})

		g.It("with a valid source struct with empty marlow field column config", func() {
			scaffold.source = strings.NewReader(`
			package marlowt
			type Author struct {
				Title 				string
				IgnoredColumn string ` + "`marlow:\"\"`" + `
			}`)
			g.Assert(scaffold.error()).Equal(nil)
		})

		g.It("with a valid source struct with explicit exclusions of certain columns", func() {
			scaffold.source = strings.NewReader(`
			package marlowt
			type Author struct {
				Title 				string
				IgnoredColumn string ` + "`marlow:\"column=-\"`" + `
			}`)
			g.Assert(scaffold.error()).Equal(nil)
		})

		g.It("errors during copy if duplicate column names", func() {
			scaffold.source = strings.NewReader(`
			package marlowt
			type Author struct {
				Title 				string
				IgnoredColumn string ` + "`marlow:\"column=dupe\"`" + `
				OtherColumn string ` + "`marlow:\"column=dupe\"`" + `
			}`)
			g.Assert(scaffold.error() == nil).Equal(false)
		})

		g.It("errors during copy if invalid column name characters", func() {
			scaffold.source = strings.NewReader(`
				package marlowt
				type Author struct {
					Title 			string
					MiddleName  string ` + "`marlow:\"column=b@dName\"`" + `
			}`)
			g.Assert(scaffold.error() == nil).Equal(false)
		})

		g.It("errors during copy if slice field type", func() {
			scaffold.source = strings.NewReader(`
				package marlowt
				type Author struct {
					Title 			string
					MiddleName  string
					SliceColumn []string ` + "`marlow:\"column=dupe\"`" + `
			}`)
			g.Assert(scaffold.error() == nil).Equal(false)
		})

		g.It("does not produce anything if all features are disabled", func() {
			scaffold.source = strings.NewReader(`
			package main

			type Author struct {
				table bool ` + "`marlow:\"updateable=false&createable=false&deletable=false&queryable=false\"`" + `
				Name  string
			}`)
			g.Assert(scaffold.error()).Equal(nil)
			r := io.MultiReader(strings.NewReader("package main\n\n"), scaffold.output)
			f, e := parser.ParseFile(token.NewFileSet(), "no-op.go", r, parser.DeclarationErrors)
			t.Logf("%s", scaffold.output.String())
			g.Assert(e).Equal(nil)
			g.Assert(len(f.Decls)).Equal(0)
		})

		g.It("errors during copy if duplicate column names (with other valid fields)", func() {
			scaffold.source = strings.NewReader(`
			package marlowt
			type Author struct {
				Title 				string
				MiddleName 		string
				IgnoredColumn string ` + "`marlow:\"column=dupe\"`" + `
				OtherColumn 	string ` + "`marlow:\"column=dupe\"`" + `
			}`)
			g.Assert(scaffold.error() == nil).Equal(false)
		})

	})
}
