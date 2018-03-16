package marlow

import "bytes"
import "strings"
import "testing"
import "go/token"
import "go/parser"
import "io/ioutil"
import "github.com/franela/goblin"

func Test_Reader(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("NewReaderFromFile", func() {
		g.It("returns an error if unable to open the provided file name", func() {
			_, e := NewReaderFromFile("not-exists")
			g.Assert(e == nil).Equal(false)
		})

		g.It("opens the file and compiles it if it exists", func() {
			f, e := ioutil.TempFile("", "marlow-reader-test")
			g.Assert(e).Equal(nil)
			_, openErr := NewReaderFromFile(f.Name())
			g.Assert(openErr).Equal(nil)
		})
	})

	g.Describe("Compile", func() {
		var output *bytes.Buffer

		g.BeforeEach(func() {
			output = new(bytes.Buffer)
		})

		g.It("fails if the provided source is invalid golang source", func() {
			e := Compile(output, strings.NewReader("}{"))
			g.Assert(e == nil).Equal(false)
		})

		g.It("skips the source if the ignore directive comment is seen", func() {
			source := strings.NewReader(`
			package marlowt
			// marlow:ignore
			type User struct {
				Name string
			}
			`)
			e := Compile(output, source)
			g.Assert(e).Equal(nil)
			g.Assert(output.Len()).Equal(0)
		})

		g.It("succeeds if the provided input is valid golang source", func() {
			source := strings.NewReader(`
			package marlowt

			type Construct struct {
				Name string
			}

			func (c *Construct) String() string {
				return "hello world"
			}
			`)
			e := Compile(output, source)
			g.Assert(e).Equal(nil)
			ts := token.NewFileSet()
			_, e = parser.ParseFile(ts, "", output, parser.AllErrors)
			g.Assert(e).Equal(nil)
		})

		g.It("succeeds if the provided input is valid golang source w/ marlow structs", func() {
			source := strings.NewReader(`
			package marlowt

			type Construct struct {
				table string ` + "`marlow:\"name=constructs_table\"`" + `
				Name string ` + "`marlow:\"column=name\"`" + `
			}
			`)
			e := Compile(output, source)
			g.Assert(e).Equal(nil)
			ts := token.NewFileSet()
			_, e = parser.ParseFile(ts, "", output, parser.AllErrors)
			g.Assert(e).Equal(nil)
		})

		g.It("correctly generates imports in the output when struct fields use imported type", func() {
			source := strings.NewReader(`
			package marlowt

			import "database/sql"

			type Construct struct {
				table string ` + "`marlow:\"name=constructs_table\"`" + `
				Name string ` + "`marlow:\"column=name\"`" + `
				ForeignID sql.NullInt64 ` + "`marlow:\"column=foreign\"`" + `
			}
			`)
			e := Compile(output, source)
			g.Assert(e).Equal(nil)
			ts := token.NewFileSet()
			_, e = parser.ParseFile(ts, "", output, parser.AllErrors)
			g.Assert(e).Equal(nil)
		})

		g.It("returns an error if a field is mis-configured", func() {
			source := strings.NewReader(`
			package marlowt

			type Construct struct {
				table string ` + "`marlow:\"tableName=&*@#@\"`" + `
				Name string ` + "`marlow:\"column=name\"`" + `
			}
			`)
			e := Compile(output, source)
			g.Assert(e.Error()).Equal("invalid-table")
		})

	})
}
