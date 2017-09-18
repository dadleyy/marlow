package marlow

import "io"
import "log"
import "fmt"
import "bytes"
import "strings"
import "testing"
import "net/url"
import "go/token"
import "go/parser"
import "github.com/franela/goblin"

func Test_goWriter(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("goWriter", func() {
		var b *bytes.Buffer
		var w *goWriter

		g.BeforeEach(func() {
			b = new(bytes.Buffer)
			w = &goWriter{log.New(b, "", 0)}
			io.Copy(b, strings.NewReader("package something\n"))
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

		g.It("support io.Writer interface via Write([]byte) (int, error)", func() {
			amt, e := w.Write([]byte("hello"))
			g.Assert(amt).Equal(5)
			g.Assert(e).Equal(nil)
		})

		g.Describe("withFunc", func() {

			g.It("returns the error that was returned from the inner func", func() {
				e := w.withFunc("myFunc", nil, nil, func(url.Values) error {
					return fmt.Errorf("bad-error")
				})

				g.Assert(e.Error()).Equal("bad-error")
			})

			g.It("formats the function correctly without any args or returns", func() {
				w.withFunc("someFunc", nil, nil, nil)
				_, e := parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
				g.Assert(e).Equal(nil)
			})

			g.It("formats the function correctly with some args but no returns", func() {
				args := []funcParam{
					{typeName: "string"},
					{paramName: "something"},
				}

				w.withFunc("someFunc", args, nil, nil)
				_, e := parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
				g.Assert(e).Equal(nil)
			})

			g.It("formats the function correctly with no args but some returns", func() {
				returns := []string{"error"}
				w.withFunc("someFunc", nil, returns, nil)
				_, e := parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
				g.Assert(e).Equal(nil)
			})
		})
	})
}
