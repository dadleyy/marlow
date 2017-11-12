package marlow

import "io"
import "fmt"
import "sort"
import "net/url"
import "strings"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type createableSymbolList struct {
	RecordParam              string
	QueryBuffer              string
	RowValueString           string
	StatementPlaceholderList string
	StatementValueList       string
	Statement                string
	StatementError           string
	SingleRecord             string
	ExecResult               string
	ExecError                string
	AffectedResult           string
	AffectedError            string
}

// newCreateableGenerator returns a reader that will generate a record store's creation api.
func newCreateableGenerator(record marlowRecord) io.Reader {
	pr, pw := io.Pipe()
	methodName := fmt.Sprintf("Create%s", inflector.Pluralize(record.name()))

	symbols := createableSymbolList{
		RecordParam:              "_records",
		QueryBuffer:              "_query",
		RowValueString:           "_placeholders",
		StatementPlaceholderList: "_placeholderList",
		StatementValueList:       "_valueList",
		Statement:                "_statement",
		StatementError:           "_e",
		SingleRecord:             "_record",
		ExecResult:               "_result",
		ExecError:                "_execError",
		AffectedResult:           "_affectedResult",
		AffectedError:            "_affectedError",
	}

	params := []writing.FuncParam{
		{Symbol: symbols.RecordParam, Type: fmt.Sprintf("...%s", record.name())},
	}

	returns := []string{
		"int64",
		"error",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)

		gosrc.Comment("[marlow] createable")

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			gosrc.WithIf("len(%s) == 0", func(url.Values) error {
				gosrc.Println("return 0, nil")
				return nil
			}, symbols.RecordParam)

			columnList := make([]string, 0, len(record.fields))
			placeholders := make([]string, 0, len(record.fields))
			fieldLookup := make(map[string]string, len(record.fields))

			for field, c := range record.fields {
				columnName := c.Get(constants.ColumnConfigOption)

				if c.Get(constants.ColumnAutoIncrementFlag) != "" {
					continue
				}

				columnList = append(columnList, columnName)
				placeholders = append(placeholders, "?")
				fieldLookup[columnName] = field
			}

			sort.Strings(columnList)

			gosrc.Println("%s := make([]string, 0, len(%s))", symbols.StatementPlaceholderList, symbols.RecordParam)
			gosrc.Println("%s := make([]interface{}, 0, len(%s))", symbols.StatementValueList, symbols.RecordParam)

			gosrc.WithIter("_, %s := range %s", func(url.Values) error {
				gosrc.Println("%s := \"(%s)\"", symbols.RowValueString, strings.Join(placeholders, ", "))
				fieldReferences := make([]string, 0, len(columnList))

				for _, columnName := range columnList {
					field := fieldLookup[columnName]
					fieldReferences = append(fieldReferences, fmt.Sprintf("%s.%s", symbols.SingleRecord, field))
				}

				gosrc.Println(
					"%s = append(%s, %s)",
					symbols.StatementValueList,
					symbols.StatementValueList,
					strings.Join(fieldReferences, ","),
				)

				gosrc.Println(
					"%s = append(%s, %s)",
					symbols.StatementPlaceholderList,
					symbols.StatementPlaceholderList,
					symbols.RowValueString,
				)
				return nil
			}, symbols.SingleRecord, symbols.RecordParam)

			gosrc.Println("%s := new(bytes.Buffer)", symbols.QueryBuffer)

			gosrc.Println(
				"fmt.Fprintf(%s, \"INSERT INTO %s ( %s ) VALUES %%s;\", strings.Join(%s, \", \"))\n",
				symbols.QueryBuffer,
				record.table(),
				strings.Join(columnList, ", "),
				symbols.StatementPlaceholderList,
			)

			gosrc.Println(
				"%s, %s := %s.Prepare(%s.String())",
				symbols.Statement,
				symbols.StatementError,
				scope.Get("receiver"),
				symbols.QueryBuffer,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.StatementError)
				return nil
			}, symbols.StatementError)

			gosrc.Println("defer %s.Close()\n", symbols.Statement)

			gosrc.Println(
				"%s, %s := %s.Exec(%s...)",
				symbols.ExecResult,
				symbols.ExecError,
				symbols.Statement,
				symbols.StatementValueList,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.ExecError)
				return nil
			}, symbols.ExecError)

			gosrc.Println("%s, %s := %s.RowsAffected()", symbols.AffectedResult, symbols.AffectedError, symbols.ExecResult)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.AffectedError)
				return nil
			}, symbols.AffectedError)

			gosrc.Println("return %s, nil", symbols.AffectedResult)
			return nil
		})

		if e == nil {
			record.registerImports("fmt", "bytes", "strings")
		}

		pw.CloseWithError(e)
	}()

	return pr
}
