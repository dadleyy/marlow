package marlow

import "io"
import "sync"
import "bytes"
import "testing"
import "net/url"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/marlow/constants"

type createableTestScaffold struct {
	buffer *bytes.Buffer

	imports chan string
	record  url.Values
	fields  map[string]url.Values

	received map[string]bool
	closed   bool
	wg       *sync.WaitGroup
}

func (s *createableTestScaffold) g() io.Reader {
	return newCreateableGenerator(s.record, s.fields, s.imports)
}

func Test_Createable(t *testing.T) {
	g := goblin.Goblin(t)

	var scaffold *createableTestScaffold

	g.Describe("createable feature generator test suite", func() {

		g.BeforeEach(func() {
			scaffold = &createableTestScaffold{
				buffer:   new(bytes.Buffer),
				wg:       &sync.WaitGroup{},
				imports:  make(chan string),
				record:   make(url.Values),
				fields:   make(map[string]url.Values),
				received: make(map[string]bool),
				closed:   false,
			}

			scaffold.wg.Add(1)

			go func() {
				for i := range scaffold.imports {
					scaffold.received[i] = true
				}
				scaffold.wg.Done()
			}()
		})

		g.AfterEach(func() {
			if scaffold.closed != false {
				return
			}

			close(scaffold.imports)
			scaffold.wg.Wait()
		})

		g.Describe("with a valid record config", func() {

			g.BeforeEach(func() {
				scaffold.record.Set(constants.RecordNameConfigOption, "Author")
				scaffold.record.Set(constants.TableNameConfigOption, "authors")
				scaffold.record.Set(constants.UpdateFieldMethodPrefixConfigOption, "Update")
				scaffold.record.Set(constants.StoreNameConfigOption, "AuthorStore")

				scaffold.fields["ID"] = url.Values{
					"type": []string{"int"},
				}

				scaffold.fields["Name"] = url.Values{
					"type": []string{"string"},
				}

				scaffold.fields["UniversityID"] = url.Values{
					"type": []string{"sql.NullInt64"},
				}
			})

			g.It("generates valid golang", func() {
				_, e := io.Copy(scaffold.buffer, scaffold.g())
				g.Assert(e).Equal(nil)
			})
		})
	})
}
