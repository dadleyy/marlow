package marlow

import "io"
import "sync"
import "bytes"
import "testing"
import "net/url"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type updateableTestScaffold struct {
	buffer   *bytes.Buffer
	imports  chan string
	methods  chan writing.FuncDecl
	record   url.Values
	fields   map[string]url.Values
	received map[string]bool
	closed   bool
	wg       *sync.WaitGroup
}

func (s *updateableTestScaffold) close() {
	if s == nil || s.closed == true {
		return
	}

	s.closed = true
	close(s.methods)
	close(s.imports)
	s.wg.Wait()
}

func (s *updateableTestScaffold) g() io.Reader {
	record := marlowRecord{
		config:        s.record,
		fields:        s.fields,
		importChannel: s.imports,
		storeChannel:  s.methods,
	}
	return newUpdateableGenerator(record)
}

func Test_Updateable(t *testing.T) {
	g := goblin.Goblin(t)

	var scaffold *updateableTestScaffold

	g.Describe("updateable feature generator test suite", func() {

		g.BeforeEach(func() {
			scaffold = &updateableTestScaffold{
				buffer:   new(bytes.Buffer),
				imports:  make(chan string),
				methods:  make(chan writing.FuncDecl),
				record:   make(url.Values),
				fields:   make(map[string]url.Values),
				received: make(map[string]bool),
				closed:   false,
				wg:       &sync.WaitGroup{},
			}

			scaffold.wg.Add(2)

			go func() {
				for range scaffold.methods {
				}
				scaffold.wg.Done()
			}()

			go func() {
				for i := range scaffold.imports {
					scaffold.received[i] = true
				}
				scaffold.wg.Done()
			}()
		})

		g.AfterEach(func() {
			scaffold.close()
		})

		g.Describe("with a record config that has nullable fields", func() {

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
