package marlow

import "testing"
import "go/types"
import "github.com/franela/goblin"

func Test_getTypeInfo(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("getTypeInfo", func() {
		g.It("returns a numeric type for time.Time", func() {
			v := getTypeInfo("time.Time")
			g.Assert(v & types.IsNumeric).Equal(v)
		})

		g.It("returns a boolean type for bools", func() {
			v := getTypeInfo("bool")
			g.Assert(v & types.IsBoolean).Equal(v)
		})

		g.It("returns a string type for strings", func() {
			v := getTypeInfo("string")
			g.Assert(v & types.IsString).Equal(v)
		})

		g.It("returns a numeric type for ints", func() {
			v := getTypeInfo("int")
			g.Assert(v & types.IsNumeric).Equal(v)
		})
	})
}
