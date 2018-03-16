package marlow

import "bytes"
import "testing"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/marlow/writing"

func Test_LogWriter(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("LogWriter test suite", func() {
		var output *bytes.Buffer
		var writer *logWriter

		g.BeforeEach(func() {
			output = new(bytes.Buffer)
			writer = &logWriter{receiver: "testing", output: writing.NewGoWriter(output)}
		})

		g.It("writes out no method calls without values", func() {
			writer.AddLog()
			g.Assert(output.Len()).Equal(0)
		})

		g.It("writes out a single value using fmt.Fprintf", func() {
			writer.AddLog("hello")
			g.Assert(output.String()).Equal("fmt.Fprintf(testing.logger,\"[marlow]  %v\\n\",hello)\n")
		})

		g.It("writes out multiple values values using fmt.Fprintf", func() {
			writer.AddLog("hello", "bye")
			g.Assert(output.String()).Equal("fmt.Fprintf(testing.logger,\"[marlow]  %v | %v\\n\",hello,bye)\n")
		})
	})
}
