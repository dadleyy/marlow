package marlow

import "io"
import "log"
import "fmt"
import "bytes"
import "strings"
import "net/url"
import "github.com/gedex/inflector"

// tableSource represents a single table
type tableSource struct {
	recordName string
	config     url.Values
	fields     map[string]url.Values
}

func (t *tableSource) WriteTo(destination io.Writer) (int64, error) {
	buffered := new(bytes.Buffer)

	writer := goWriter{
		Logger: log.New(buffered, "", 0),
	}

	tableName := t.config.Get("tableName")
	storeName := t.storeName()

	if tableName == "" {
		tableName = strings.ToLower(inflector.Pluralize(t.recordName))
	}

	writer.Printf("// record: %s\n// table: %s", t.recordName, tableName)
	writer.Println()

	writer.withStruct(storeName, func(url.Values) error {
		writer.Printf("*sql.DB")
		return nil
	})

	qParams := []funcParam{
		funcParam{paramName: "sqlQuery", typeName: "string"},
		funcParam{paramName: "args", typeName: "...interface{}"},
	}

	qReturns := []string{"*sql.Rows", "error"}

	writer.withMetod("q", t.storeName(), qParams, qReturns, func(scope url.Values) error {
		receiver := scope.Get("receiver")
		condition := fmt.Sprintf("%s.DB == nil || %s.Ping() != nil", receiver, receiver)

		writer.withIf(condition, func(url.Values) error {
			writer.Println("return nil, fmt.Errorf(\"\")")
			return nil
		})

		writer.Printf("return %s.Query(sqlQuery, args...)", receiver)
		return nil
	})

	if qf := t.queryableFields(); len(qf) == 0 {
		return io.Copy(destination, buffered)
	}

	if _, e := io.Copy(&writer, recordFinderGenerator(t)); e != nil {
		return 0, e
	}

	return io.Copy(destination, buffered)
}

func (t *tableSource) tableName() string {
	name := t.config.Get("tableName")

	if name != "" {
		return name
	}

	return strings.ToLower(inflector.Pluralize(t.recordName))
}

func (t *tableSource) storeName() string {
	singular := inflector.Singularize(t.recordName)
	return fmt.Sprintf("%sStore", singular)
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
