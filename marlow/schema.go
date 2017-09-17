package marlow

import "io"
import "log"
import "fmt"
import "bytes"
import "net/url"
import "github.com/gedex/inflector"

// tableSource represents a single table
type tableSource struct {
	recordName string
	fields     map[string]url.Values
}

func (t *tableSource) WriteTo(destination io.Writer) (int64, error) {
	buffered := new(bytes.Buffer)

	writer := Writer{
		Logger: log.New(buffered, "", 0),
	}

	writer.Printf("// %s", t.recordName)

	singularName := inflector.Singularize(t.recordName)
	storeName := fmt.Sprintf("%sStore", singularName)

	writer.withStruct(storeName, func(w *Writer) error {
		w.Printf("*sql.DB")
		return nil
	})

	if qf := t.queryableFields(); len(qf) >= 1 {
		queryStruct := fmt.Sprintf("%sQuery", singularName)

		writer.withStruct(queryStruct, func(w *Writer) error {
			for name, config := range t.queryableFields() {
				writer.Printf("%s []%s", name, config.Get("type"))
			}

			return nil
		})

		findName := fmt.Sprintf("Find%s", inflector.Pluralize(t.recordName))

		findParams := make(map[string]string)

		findParams["q"] = queryStruct

		findReturns := []string{
			fmt.Sprintf("[]*%s", singularName),
			"error",
		}

		writer.withMetod(findName, storeName, findParams, findReturns, func(w *Writer) error {
			w.Printf("return nil, fmt.Errorf(\"\")")
			return nil
		})
	}

	return io.Copy(destination, buffered)
}

func (t *tableSource) queryableFields() map[string]url.Values {
	result := make(map[string]url.Values)

	for name, config := range t.fields {
		if config.Get("queryable") == "false" {
			continue
		}

		result[name] = config
	}

	return result
}

// schema represents a collection of record names and their fields.
type schema map[string]*tableSource

func (s *schema) WriteTo(destination io.Writer) (int64, error) {
	buffered := new(bytes.Buffer)

	for _, table := range *s {
		if _, e := table.WriteTo(buffered); e != nil {
			return 0, e
		}
	}

	return io.Copy(destination, buffered)
}

func (s *schema) dependencies() []string {
	deps := []string{"fmt", "database/sql"}
	return deps
}

func (s *schema) reader() io.Reader {
	pr, pw := io.Pipe()

	go func() {
		_, e := s.WriteTo(pw)
		pw.CloseWithError(e)
	}()

	return pr
}
