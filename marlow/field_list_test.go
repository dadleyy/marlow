package marlow

import "sort"
import "testing"

import "github.com/franela/goblin"

func Test_FieldList(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("fieldList test suite", func() {

		g.It("sorts correctly", func() {
			l := fieldList{
				{name: "z", column: "a"},
				{name: "a", column: "z"},
			}
			sort.Sort(l)
			g.Assert(l[0].name).Equal("a")
			g.Assert(l[1].name).Equal("z")
		})

	})
}
