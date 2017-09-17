package marlow

import "bytes"
import "testing"
import "net/url"
import "github.com/franela/goblin"

func Test_Schema(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("tableSource", func() {
		var source *tableSource
		var dest *bytes.Buffer

		g.BeforeEach(func() {
			source = &tableSource{
				recordName: "TestSource",
				fields:     make(map[string]url.Values),
			}

			dest = new(bytes.Buffer)
		})

		g.It("successfully writes the basic dependencies of the table source", func() {
			_, e := source.WriteTo(dest)
			g.Assert(e).Equal(nil)
		})
	})
}
