package writing

import "io"
import "log"
import "fmt"
import "bytes"
import "strings"
import "testing"
import "net/url"
import "go/token"
import "go/parser"
import "go/format"
import "github.com/franela/goblin"

type parseableBuffer struct {
	*bytes.Buffer
}

func (b *parseableBuffer) ParseError() error {
	formatted, e := format.Source(b.Bytes())

	if e != nil {
		return e
	}

	if _, e := parser.ParseFile(token.NewFileSet(), "", bytes.NewBuffer(formatted), parser.AllErrors); e != nil {
		return e
	}

	return nil
}

func Test_goWriter(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("goWriter", func() {
		var b *parseableBuffer
		var w *goWriter

		g.BeforeEach(func() {
			b = &parseableBuffer{new(bytes.Buffer)}
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
	})

	g.Describe("writer test suite", func() {
		var b *parseableBuffer
		var w GoWriter

		g.BeforeEach(func() {
			b = &parseableBuffer{new(bytes.Buffer)}
			w = NewGoWriter(b)
			io.Copy(b, strings.NewReader("package something\n\n"))
		})

		g.Describe("Comment", func() {
			g.It("returns an invalid condition error if conditional is too short", func() {
				w.Comment("hello world")
				g.Assert(strings.Contains(b.String(), "// hello world")).Equal(true)
			})
		})

		g.Describe("WriteImport", func() {

			g.It("prepends the package keyword", func() {
				w.WriteImport("mypack")
				g.Assert(strings.Contains(b.String(), "import \"mypack\"")).Equal(true)
			})

		})

		g.Describe("WritePakcage", func() {

			g.It("prepends the package keyword", func() {
				w.WritePackage("mypack")
				g.Assert(strings.Contains(b.String(), "mypack")).Equal(true)
			})

		})

		g.Describe("WithIter", func() {
			g.It("returns an invalid condition error if conditional is too short", func() {
				e := w.WithIter("", nil)
				g.Assert(e.Error()).Equal("invalid-condition")
			})

			g.It("returns the error returned from the block", func() {
				e := w.WithIter("e := nil; e == nil", func(url.Values) error {
					return fmt.Errorf("bad-write")
				})
				g.Assert(e.Error()).Equal("bad-write")
			})

			g.It("returns invalid golang if not wrapped within another decl", func() {
				e := w.WithIter("true", nil)
				g.Assert(e).Equal(nil)
				g.Assert(b.ParseError() == nil).Equal(false)
			})

			g.It("returns valid golang if wrapped within another decl", func() {
				e := w.WithFunc("helloWorld", nil, nil, func(url.Values) error {
					return w.WithIter("true", nil)
				})
				g.Assert(e).Equal(nil)
				g.Assert(b.ParseError()).Equal(nil)
			})
		})

		g.Describe("WithIf", func() {
			g.It("returns an invalid condition error if conditional is too short", func() {
				e := w.WithIf("", nil)
				g.Assert(e.Error()).Equal("invalid-condition")
			})

			g.It("returns the error returned from the block", func() {
				e := w.WithIf("e := nil; e == nil", func(url.Values) error {
					return fmt.Errorf("bad-write")
				})
				g.Assert(e.Error()).Equal("bad-write")
			})

			g.It("returns invalid golang if not wrapped within another decl", func() {
				e := w.WithIf("true", nil)
				g.Assert(e).Equal(nil)
				g.Assert(b.ParseError() == nil).Equal(false)
			})

			g.It("returns valid golang if wrapped within another decl", func() {
				e := w.WithFunc("helloWorld", nil, nil, func(url.Values) error {
					return w.WithIf("true", nil)
				})
				g.Assert(e).Equal(nil)
				g.Assert(b.ParseError()).Equal(nil)
			})
		})

		g.Describe("WithStruct", func() {
			g.It("returns an invalid receiver error if type name is too short", func() {
				e := w.WithStruct("", nil)
				g.Assert(e.Error()).Equal("invalid-name")
			})

			g.It("returns the error returned by the block", func() {
				e := w.WithStruct("myFunc", func(url.Values) error {
					return fmt.Errorf("bad-error")
				})
				g.Assert(e.Error()).Equal("bad-error")
			})

			g.It("successfully wrote valid golang if no error", func() {
				e := w.WithStruct("myFunc", nil)
				g.Assert(e).Equal(nil)
				g.Assert(b.ParseError()).Equal(nil)
			})
		})

		g.Describe("WithMethod", func() {
			g.It("returns an invalid receiver error if type name is too short", func() {
				e := w.WithMethod("myFunc", "", nil, nil, nil)
				g.Assert(e.Error()).Equal("invalid-receiver")
			})

			g.It("returns the error returned by the block", func() {
				e := w.WithMethod("myFunc", "myType", nil, nil, func(url.Values) error {
					return fmt.Errorf("bad-error")
				})
				g.Assert(e.Error()).Equal("bad-error")
			})

			g.It("successfully wrote valid golang if no error, no params or returns", func() {
				e := w.WithMethod("myFunc", "myType", nil, nil, nil)
				g.Assert(e).Equal(nil)
				_, e = parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
				g.Assert(e).Equal(nil)
			})

			g.It("successfully wrote valid golang if no error and no returns", func() {
				e := w.WithMethod("myFunc", "myType", []FuncParam{
					{Type: "string", Symbol: "whoa"},
				}, nil, nil)
				g.Assert(e).Equal(nil)
				_, e = parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
				g.Assert(e).Equal(nil)
			})

		})

		g.Describe("WithFunc", func() {

			g.It("returns the error that was returned from the inner func", func() {
				e := w.WithFunc("myFunc", nil, nil, func(url.Values) error {
					return fmt.Errorf("bad-error")
				})

				g.Assert(e.Error()).Equal("bad-error")
			})

			g.It("formats the function correctly without any args or returns", func() {
				w.WithFunc("someFunc", nil, nil, nil)
				_, e := parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
				g.Assert(e).Equal(nil)
			})

			g.It("formats the function correctly with some args but no returns", func() {
				args := []FuncParam{
					{Type: "string"},
					{Symbol: "something"},
				}

				w.WithFunc("someFunc", args, nil, nil)
				_, e := parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
				g.Assert(e).Equal(nil)
			})

			g.It("formats the function correctly with no args but some returns", func() {
				returns := []string{"error"}
				w.WithFunc("someFunc", nil, returns, nil)
				_, e := parser.ParseFile(token.NewFileSet(), "", b, parser.AllErrors)
				g.Assert(e).Equal(nil)
			})
		})
	})
}
