package writing

import "fmt"
import "testing"
import "github.com/franela/goblin"

func Test_Types(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("mapListToWrappedCommaList", func() {
		g.It("wraps and delimits a list with the appropriate characters", func() {
			v := mapListToWrappedCommaList([]string{"hello", "world"}, "x")
			g.Assert(v).Equal("xhellox,xworldx")
		})

		g.It("wraps a single-item list with the appropriate characters", func() {
			v := mapListToWrappedCommaList([]string{"hello"}, "x")
			g.Assert(v).Equal("xhellox")
		})

		g.It("returns an empty string for empty lists", func() {
			v := mapListToWrappedCommaList([]string{}, "x")
			g.Assert(v).Equal("")
		})
	})

	g.Describe("StringSliceLiteral", func() {
		var l StringSliceLiteral

		g.It("formats as a single quoted list separated by commas when used with %s", func() {
			l = []string{"hello", "world"}
			v := fmt.Sprintf("%s", l)
			g.Assert(v).Equal("[]string{\"hello\",\"world\"}")
		})
	})

	g.Describe("SingleQuotedStringList", func() {
		var l SingleQuotedStringList

		g.It("formats as a single quoted list separated by commas when used with %s", func() {
			l = []string{"hello", "world"}
			v := fmt.Sprintf("%s", l)
			g.Assert(v).Equal("'hello','world'")
		})
	})
}
