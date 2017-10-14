package features

import "io"
import "fmt"
import "net/url"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

func updater(record url.Values, name string, config url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()
	blueprint := blueprint{record: record}
	recordName := record.Get(constants.RecordNameConfigOption)
	methodName := fmt.Sprintf("%s%s%s", record.Get(constants.UpdateFieldMethodPrefixConfigOption), recordName, name)
	tableName, columnName := record.Get(constants.TableNameConfigOption), config.Get(constants.ColumnConfigOption)
	storeName := record.Get(constants.StoreNameConfigOption)

	symbols := map[string]string{
		"VALUE_PARAM":     "_newValue",
		"BLUEPRINT_PARAM": "_blueprint",
		"QUERY_BUFFER":    "_query",
		"QUERY_RESULT":    "_queryResult",
		"SET_VALUE":       "_setValue",
		"ROW_COUNT":       "_rowCount",
	}

	params := []writing.FuncParam{
		{Type: config.Get("type"), Symbol: symbols["VALUE_PARAM"]},
		{Type: fmt.Sprintf("*%s", blueprint.Name()), Symbol: symbols["BLUEPRINT_PARAM"]},
	}

	if config.Get("type") == "sql.NullInt64" {
		params[0].Type = fmt.Sprintf("*%s", config.Get("type"))
	}

	returns := []string{
		"int64",
		"error",
		"string",
	}

	replacementFormatString := "'%s'"

	switch config.Get("type") {
	case "sql.NullInt64":
		replacementFormatString = "%v"
	case "int":
		replacementFormatString = "%d"
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)
		gosrc.Comment("[marlow] updater method for %s", name)

		e := gosrc.WithMethod(methodName, storeName, params, returns, func(scope url.Values) error {
			gosrc.Println("%s := bytes.NewBufferString(\"UPDATE %s\")", symbols["QUERY_BUFFER"], tableName)

			gosrc.Println(
				"%s := fmt.Sprintf(\"%s\", %s)",
				symbols["SET_VALUE"],
				replacementFormatString,
				symbols["VALUE_PARAM"],
			)

			switch config.Get("type") {
			case "sql.NullInt64":
				gosrc.WithIf("%s == nil || !%s.Valid", func(url.Values) error {
					gosrc.Println("%s = \"NULL\"", symbols["SET_VALUE"])
					return nil
				}, symbols["VALUE_PARAM"], symbols["VALUE_PARAM"])

				gosrc.WithIf("%s != nil && %s.Valid", func(url.Values) error {
					gosrc.Println("%s = fmt.Sprintf(\"%%d\", %s.Int64)", symbols["SET_VALUE"], symbols["VALUE_PARAM"])
					return nil
				}, symbols["VALUE_PARAM"], symbols["VALUE_PARAM"])
			}

			gosrc.Println(
				"fmt.Fprintf(%s, \" SET %s = %%s\", %s)",
				symbols["QUERY_BUFFER"],
				columnName,
				symbols["SET_VALUE"],
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("fmt.Fprintf(%s, \" %%s\", %s)", symbols["QUERY_BUFFER"], symbols["BLUEPRINT_PARAM"])
				return nil
			}, symbols["BLUEPRINT_PARAM"])

			// Write the query execution statement.
			gosrc.Println(
				"%s, e := %s.e(%s.String() + \";\")",
				symbols["QUERY_RESULT"],
				scope.Get("receiver"),
				symbols["QUERY_BUFFER"],
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, e, %s.String()", symbols["QUERY_BUFFER"])
				return nil
			}, "e")

			gosrc.Println("%s, e := %s.RowsAffected()", symbols["ROW_COUNT"], symbols["QUERY_RESULT"])

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, e, %s.String()", symbols["QUERY_BUFFER"])
				return nil
			}, "e")

			gosrc.Println(
				"return %s, nil, %s.String()",
				symbols["ROW_COUNT"],
				symbols["QUERY_BUFFER"],
			)
			return nil
		})

		if e == nil {
			imports <- "fmt"
			imports <- "bytes"
		}

		pw.CloseWithError(e)
	}()

	return pr
}

// NewUpdateableGenerator is responsible for generating updating store methods.
func NewUpdateableGenerator(record url.Values, fields map[string]url.Values, imports chan<- string) io.Reader {
	readers := make([]io.Reader, 0, len(fields))

	for name, config := range fields {
		readers = append(readers, updater(record, name, config, imports))
	}

	return io.MultiReader(readers...)
}
