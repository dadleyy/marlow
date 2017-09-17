package marlow

import "io"
import "bytes"
import "go/ast"
import "strings"
import "testing"
import "net/url"
import "go/token"
import "go/parser"
import "github.com/franela/goblin"

type finderDest struct {
	*bytes.Buffer
}

func (d *finderDest) parsedAst() (*ast.File, error) {
	return parser.ParseFile(token.NewFileSet(), "", d.Buffer, parser.AllErrors)
}

func Test_copyRecordFinder(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("copyRecordFinder", func() {
		var source *tableSource
		var dest *finderDest
		var config url.Values
		var fields map[string]url.Values

		g.BeforeEach(func() {
			config = make(url.Values)
			fields = make(map[string]url.Values)

			source = &tableSource{
				recordName: "Goblin",
				config:     config,
				fields:     fields,
			}

			dest = &finderDest{new(bytes.Buffer)}

			io.Copy(dest, strings.NewReader("package marlowt"))
		})

		g.It("successfully creates valid golang source", func() {
			e := copyRecordFinder(dest, source)
			g.Assert(e).Equal(nil)
			_, e = parser.ParseFile(token.NewFileSet(), "", dest, parser.AllErrors)
			g.Assert(e).Equal(nil)
		})

		g.Describe("with a source that has queryable fields", func() {
			g.BeforeEach(func() {
				idField := make(url.Values)
				idField.Set("column", "system_id")
				idField.Set("type", "uint")

				nameField := make(url.Values)
				nameField.Set("type", "string")

				fields["ID"] = idField
				fields["Name"] = nameField
			})

			g.It("produces a valid query struct", func() {
				e := copyRecordFinder(dest, source)
				g.Assert(e).Equal(nil)
				a, e := dest.parsedAst()
				g.Assert(e).Equal(nil)
				g.Assert(len(a.Decls) > 0).Equal(true)
			})
		})
	})
}
