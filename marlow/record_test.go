package marlow

import "sync"
import "testing"
import "github.com/franela/goblin"

func Test_Record(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("marlowRecord test suite", func() {
		var record *marlowRecord
		var imports chan string
		var received map[string]int
		var wg *sync.WaitGroup

		g.BeforeEach(func() {
			imports = make(chan string)
			record = &marlowRecord{importChannel: imports}
			received = make(map[string]int)
			wg = &sync.WaitGroup{}

			wg.Add(1)

			go func() {
				for i := range imports {
					_, dupe := received[i]

					if !dupe {
						received[i] = 1
						continue
					}

					received[i]++
				}
				wg.Done()
			}()
		})

		g.It("only registers received imports once", func() {
			record.registerImports("fmt", "fmt")
			close(imports)
			wg.Wait()
			g.Assert(received["fmt"]).Equal(1)
		})
	})
}
