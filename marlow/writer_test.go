package marlow

import "log"
import "bytes"
import "testing"
import "github.com/franela/goblin"

func Test_Writer(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Writer", func() {
		var b *bytes.Buffer
		var w *Writer

		g.BeforeEach(func() {
			b = new(bytes.Buffer)
			w = &Writer{log.New(b, "", 0)}
		})

		g.It("correctly formats an empty return value", func() {
			s := w.formatReturns([]string{})
			g.Assert(s).Equal("")
		})

		g.It("correctly formats a single return value", func() {
			s := w.formatReturns([]string{"hi"})
			g.Assert(s).Equal("hi")
		})

		g.It("correctly formats a multiple return value", func() {
			s := w.formatReturns([]string{"hi", "bye"})
			g.Assert(s).Equal("(hi,bye)")
		})
	})
}
