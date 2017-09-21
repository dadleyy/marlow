package marlow

import "io"
import "bytes"
import "go/ast"
import "go/token"
import "go/parser"
import "testing"
import "strings"
import "github.com/franela/goblin"

type recordReaderTestScaffold struct {
	source  io.Reader
	imports chan string
	output  *bytes.Buffer
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

func Test_RecordReader(t *testing.T) {
	g := goblin.Goblin(t)

	var scaffold *recordReaderTestScaffold

	g.Describe("newRecordReader", func() {

		g.BeforeEach(func() {
			scaffold = &recordReaderTestScaffold{
				imports: make(chan string),
				output:  new(bytes.Buffer),
			}
		})

		g.It("returns false if the root is not a valid marlow struct", func() {
			scaffold.source = strings.NewReader(`
			package marlowt

			func someFunction() {
			}
			`)
			_, ok := newRecordReader(scaffold.root(), scaffold.imports)
			g.Assert(ok).Equal(false)
		})

		g.It("returns true if the root is a valid marlow struct", func() {
			scaffold.source = strings.NewReader(`
			package marlowt

			type Book struct {
				Title string ` + "`marlow:\"\"`" + `
			}
			`)
			_, ok := newRecordReader(scaffold.root(), scaffold.imports)
			g.Assert(ok).Equal(true)
		})

		g.It("returns a reader that will error during copy if the record has an invalid tableName", func() {
			scaffold.source = strings.NewReader(`
			package marlowt

			type Book struct {
				table string ` + "`marlow:\"tableName=@@#\"`" + `
				Title string ` + "`marlow:\"column=title\"`" + `
			}
			`)
			r, ok := newRecordReader(scaffold.root(), scaffold.imports)
			g.Assert(ok).Equal(true)
			_, e := io.Copy(scaffold.output, r)
			g.Assert(e.Error()).Equal("invalid-table")
		})

	})
}
