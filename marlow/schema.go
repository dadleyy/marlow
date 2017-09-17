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

	writer := Writer{
		Logger: log.New(buffered, "", 0),
	}

	tableName := t.config.Get("tableName")

	if tableName == "" {
		tableName = strings.ToLower(inflector.Pluralize(t.recordName))
	}

	writer.Printf("// record: %s\n// table: %s", t.recordName, tableName)
	writer.Println()

	singularName := inflector.Singularize(t.recordName)
	storeName := fmt.Sprintf("%sStore", singularName)

	writer.withStruct(storeName, func(url.Values) error {
		writer.Printf("*sql.DB")
		return nil
	})

	if qf := t.queryableFields(); len(qf) >= 1 {
		queryStruct := fmt.Sprintf("%sQuery", singularName)

		writer.withStruct(queryStruct, func(url.Values) error {
			for name, config := range t.queryableFields() {
				writer.Printf("%s []%s", name, config.Get("type"))
			}

			writer.Printf("Limit uint")
			writer.Printf("Offset uint")

			return nil
		})

		findName := fmt.Sprintf("Find%s", inflector.Pluralize(t.recordName))

		findParams := make(map[string]string)

		findParams["q"] = queryStruct

		findReturns := []string{
			fmt.Sprintf("[]*%s", singularName),
			"error",
		}

		qParams := map[string]string{
			"sqlQuery": "string",
			"args":     "...interface{}",
		}

		writer.withMetod("q", storeName, qParams, []string{"*sql.Rows", "error"}, func(scope url.Values) error {
			receiver := scope.Get("receiver")
			condition := fmt.Sprintf("%s.DB == nil || %s.Ping() != nil", receiver, receiver)

			writer.withIf(condition, func(url.Values) error {
				writer.Println("return nil, fmt.Errorf(\"\")")
				return nil
			})

			writer.Printf("return %s.Query(sqlQuery, args...)", receiver)
			return nil
		})

		writer.withMetod(findName, storeName, findParams, findReturns, func(scope url.Values) error {
			writer.Printf("query := fmt.Sprintf(\"SELECT * FROM %s;\")", tableName)
			writer.Printf("results, e := %s.q(query)", scope.Get("receiver"))
			writer.Printf("out := make([]*%s, 0)", t.recordName)
			writer.Println()

			writer.withIf("e != nil", func(url.Values) error {
				writer.Printf("return nil, e")
				return nil
			})

			writer.Println("defer results.Close()")
			writer.Println("")

			writer.withIf("e := results.Err(); e != nil", func(url.Values) error {
				writer.Printf("return nil, e")
				return nil
			})

			writer.withIter("results.Next()", func(url.Values) error {
				writer.Printf("var b %s", t.recordName)

				writer.withIf("e := results.Scan(&b); e != nil", func(url.Values) error {
					writer.Printf("return nil, e")
					return nil
				})

				writer.Println("out = append(out, &b)")
				return nil
			})

			writer.Println("return out, nil")
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
