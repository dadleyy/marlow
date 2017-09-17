package marlow

import "bytes"
import "strings"
import "testing"
import "go/token"
import "go/parser"
import "github.com/franela/goblin"

func Test_Reader(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Compile", func() {
		var output *bytes.Buffer

		g.BeforeEach(func() {
			output = new(bytes.Buffer)
		})

		g.It("fails if the provided source is invalid golang source", func() {
			e := Compile(output, strings.NewReader("}{"))
			g.Assert(e == nil).Equal(false)
		})

		g.It("succeeds if the provided input is valid golang source", func() {
			source := strings.NewReader(`
			package marlowt

			type Construct struct {
				Name string
			}
			`)
			e := Compile(output, source)
			g.Assert(e).Equal(nil)
			ts := token.NewFileSet()
			_, e = parser.ParseFile(ts, "", output, parser.AllErrors)
			g.Assert(e).Equal(nil)
		})
	})
}
