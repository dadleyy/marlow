package marlow

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

func updater(record marlowRecord, fieldName string, fieldConfig url.Values) io.Reader {
	pr, pw := io.Pipe()
	methodName := fmt.Sprintf(
		"%s%s%s",
		record.config.Get(constants.UpdateFieldMethodPrefixConfigOption),
		record.name(),
		fieldName,
	)
	columnName := fieldConfig.Get(constants.ColumnConfigOption)

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
		{Type: fieldConfig.Get("type"), Symbol: symbols.valueParam},
		{Type: fmt.Sprintf("*%s", record.config.Get(constants.BlueprintNameConfigOption)), Symbol: symbols.blueprintParam},
	}

	if fieldConfig.Get("type") == "sql.NullInt64" {
		params[0].Type = fmt.Sprintf("*%s", fieldConfig.Get("type"))
	}

	returns := []string{
		"int64",
		"error",
		"string",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)
		gosrc.Comment("[marlow] updater method for %s", fieldName)

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			gosrc.Println(
				"%s := bytes.NewBufferString(\"UPDATE %s set %s = ?\")",
				symbols.queryString,
				record.table(),
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
			record.registerImports("fmt", "bytes")
			record.registerStoreMethod(writing.FuncDecl{
				Name:    methodName,
				Params:  params,
				Returns: returns,
			})
		}

		pw.CloseWithError(e)
	}()

	return pr
}

// newUpdateableGenerator is responsible for generating updating store methods.
func newUpdateableGenerator(record marlowRecord) io.Reader {
	readers := make([]io.Reader, 0, len(record.fields))

	for name, config := range record.fields {
		u := updater(record, name, config)
		readers = append(readers, u)
	}

	return io.MultiReader(readers...)
}
