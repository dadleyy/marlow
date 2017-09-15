package marlow

import "io"
import "fmt"
import "bytes"
import "net/url"
import "go/format"
import "github.com/gedex/inflector"

type record map[string]url.Values
type recordStore map[string]record

func (s *recordStore) dependencies() []string {
	return []string{"github.com/foo/bar"}
}

func (s *recordStore) writeTo(destination io.Writer) error {
	buffer := new(bytes.Buffer)

	for _, dependency := range s.dependencies() {
		fmt.Fprintf(buffer, "import \"%s\"\n", dependency)
	}

	fmt.Fprintln(buffer)

	for name, definition := range *s {
		recordName := inflector.Singularize(name)
		queryName := fmt.Sprintf("%sQuery", recordName)
		lookupName := fmt.Sprintf("Find%s", inflector.Pluralize(name))

		fmt.Fprintf(buffer, "type %s struct {\n", queryName)

		for field, fieldConfig := range definition {
			fieldType := fieldConfig.Get("type")
			fmt.Fprintf(buffer, "%s []%s\n", field, fieldType)
		}

		fmt.Fprintln(buffer, "}\n")

		fmt.Fprintf(buffer, "func %s(query %s) ([]%s, err) {", lookupName, queryName, name)
		fmt.Fprintf(buffer, "results := make([]%s, 0)\n", name)
		fmt.Fprintln(buffer, "return results, nil")
		fmt.Fprintln(buffer, "}\n")

		fmt.Fprintln(buffer)
	}

	formatted, e := format.Source(buffer.Bytes())

	if e != nil {
		panic(e)
		return e
	}

	_, e = destination.Write(formatted)
	return e
}

func NewGenerator(store *recordStore) io.Reader {
	reader, writer := io.Pipe()

	go func() {
		e := store.writeTo(writer)
		reader.CloseWithError(e)
		writer.CloseWithError(e)
	}()

	return reader
}
