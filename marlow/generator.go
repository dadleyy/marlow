package marlow

import "io"
import "fmt"
import "bytes"
import "net/url"
import "go/format"
import "github.com/gedex/inflector"

type record map[string]url.Values

func (r *record) queryableFields() map[string]url.Values {
	result := make(map[string]url.Values)

	for name, config := range *r {
		if config.Get("queryable") == "false" {
			continue
		}

		result[name] = config
	}

	return result
}

type recordStore map[string]record

func (s *recordStore) dependencies() []string {
	return []string{"github.com/foo/bar"}
}

func (s *recordStore) writeTo(destination io.Writer) error {
	reader, writer := io.Pipe()

	go func() {
		buffer := new(bytes.Buffer)

		for _, dependency := range s.dependencies() {
			fmt.Fprintf(buffer, "import \"%s\"\n", dependency)
		}

		fmt.Fprintln(buffer)

		for name, definition := range *s {
			recordName := inflector.Singularize(name)
			queryName := fmt.Sprintf("%sQuery", recordName)
			lookupName := fmt.Sprintf("Find%s", inflector.Pluralize(name))

			queryableFields := definition.queryableFields()

			if len(queryableFields) >= 1 {
				construct := newQueryConstruct(recordName, queryableFields)
				io.Copy(buffer, construct)
			}

			fmt.Fprintf(buffer, "func %s(query %s) ([]*%s, err) {", lookupName, queryName, recordName)
			fmt.Fprintf(buffer, "results ===:= make([]%s, 0)\n", recordName)
			fmt.Fprintln(buffer, "return results, nil")
			fmt.Fprintln(buffer, "}\n")

			fmt.Fprintln(buffer)
		}

		formatted, e := format.Source(buffer.Bytes())

		if e != nil {
			writer.CloseWithError(e)
			return
		}

		_, e = io.Copy(writer, bytes.NewBuffer(formatted))
		writer.CloseWithError(e)
	}()

	_, e := io.Copy(destination, reader)
	return e
}

func NewGenerator(store *recordStore) io.Reader {
	reader, writer := io.Pipe()

	go func() {
		e := store.writeTo(writer)
		writer.CloseWithError(e)
	}()

	return reader
}
