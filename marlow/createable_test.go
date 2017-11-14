package marlow

import "io"
import "sync"
import "bytes"
import "testing"
import "net/url"
import "github.com/franela/goblin"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type createableTestScaffold struct {
	buffer *bytes.Buffer

	imports chan string
	methods chan writing.FuncDecl

	record url.Values
	fields map[string]url.Values

	received map[string]bool
	closed   bool
	wg       *sync.WaitGroup
}

func (s *createableTestScaffold) close() {
	if s == nil || s.closed {
		return
	}

	s.closed = true
	close(s.imports)
	close(s.methods)
	s.wg.Wait()
}

func (s *createableTestScaffold) g() io.Reader {
	record := marlowRecord{
		fields:        s.fields,
		config:        s.record,
		importChannel: s.imports,
		storeChannel:  s.methods,
	}

	return newCreateableGenerator(record)
}

func Test_Createable(t *testing.T) {
	g := goblin.Goblin(t)

	var scaffold *createableTestScaffold

	g.Describe("createable feature generator test suite", func() {

		g.BeforeEach(func() {
			scaffold = &createableTestScaffold{
				buffer: new(bytes.Buffer),
				wg:     &sync.WaitGroup{},

				imports: make(chan string),
				methods: make(chan writing.FuncDecl),

				record:   make(url.Values),
				fields:   make(map[string]url.Values),
				received: make(map[string]bool),
				closed:   false,
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

			g.Describe("with a postgres record dialect", func() {
				g.BeforeEach(func() {
					scaffold.record.Set(constants.DialectConfigOption, "postgres")
				})

				g.It("returns an error without a primaryKey defined", func() {
					_, e := io.Copy(scaffold.buffer, scaffold.g())
					g.Assert(e != nil).Equal(true)
				})

				g.It("compiles successfully with a valid primaryKey defined", func() {
					scaffold.record.Set(constants.PrimaryKeyColumnConfigOption, "id")
					_, e := io.Copy(scaffold.buffer, scaffold.g())
					g.Assert(e).Equal(nil)
				})
			})
		})
	})
}
