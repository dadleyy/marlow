package features

import "io"
import "fmt"
import "net/url"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type updaterSymbols struct {
	valueParam      string
	blueprintParam  string
	queryString     string
	queryResult     string
	queryError      string
	statementResult string
	statementError  string
	rowCount        string
	rowError        string
	valueSlice      string
}

func updater(record url.Values, name string, config url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()
	blueprint := blueprint{record: record}
	recordName := record.Get(constants.RecordNameConfigOption)
	methodName := fmt.Sprintf("%s%s%s", record.Get(constants.UpdateFieldMethodPrefixConfigOption), recordName, name)
	tableName, columnName := record.Get(constants.TableNameConfigOption), config.Get(constants.ColumnConfigOption)
	storeName := record.Get(constants.StoreNameConfigOption)

	symbols := updaterSymbols{
		valueParam:      "_updates",
		blueprintParam:  "_blueprint",
		queryString:     "_queryString",
		queryResult:     "_queryResult",
		queryError:      "_queryError",
		statementResult: "_statement",
		statementError:  "_se",
		rowCount:        "_rowCount",
		rowError:        "_re",
		valueSlice:      "_values",
	}

	params := []writing.FuncParam{
		{Type: config.Get("type"), Symbol: symbols.valueParam},
		{Type: fmt.Sprintf("*%s", blueprint.Name()), Symbol: symbols.blueprintParam},
	}

	if config.Get("type") == "sql.NullInt64" {
		params[0].Type = fmt.Sprintf("*%s", config.Get("type"))
	}

	returns := []string{
		"int64",
		"error",
		"string",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)
		gosrc.Comment("[marlow] updater method for %s", name)

		e := gosrc.WithMethod(methodName, storeName, params, returns, func(scope url.Values) error {
			gosrc.Println(
				"%s := bytes.NewBufferString(\"UPDATE %s set %s = ?\")",
				symbols.queryString,
				tableName,
				columnName,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("fmt.Fprintf(%s, \" %%s\", %s)", symbols.queryString, symbols.blueprintParam)
				return nil
			}, symbols.blueprintParam)

			// Write the query execution statement.
			gosrc.Println(
				"%s, %s := %s.Prepare(%s.String() + \";\")",
				symbols.statementResult,
				symbols.statementError,
				scope.Get("receiver"),
				symbols.queryString,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s, %s.String()", symbols.statementError, symbols.queryString)
				return nil
			}, symbols.statementError)

			gosrc.Println("defer %s.Close()", symbols.statementResult)

			gosrc.Println("%s := make([]interface{}, 1)", symbols.valueSlice)
			gosrc.Println("%s[0] = %s", symbols.valueSlice, symbols.valueParam)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println(
					"%s = append(%s, %s.Values()...)",
					symbols.valueSlice,
					symbols.valueSlice,
					symbols.blueprintParam,
				)
				return nil
			}, symbols.blueprintParam)

			gosrc.Println("%s, %s := %s.Exec(%s...)",
				symbols.queryResult,
				symbols.queryError,
				symbols.statementResult,
				symbols.valueSlice,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s, %s.String()", symbols.queryError, symbols.queryString)
				return nil
			}, symbols.queryError)

			gosrc.Println("%s, %s := %s.RowsAffected()", symbols.rowCount, symbols.rowError, symbols.queryResult)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s, %s.String()", symbols.rowError, symbols.queryString)
				return nil
			}, symbols.rowError)

			gosrc.Println(
				"return %s, nil, %s.String()",
				symbols.rowCount,
				symbols.queryString,
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
