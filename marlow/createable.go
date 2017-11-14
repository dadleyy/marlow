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
	recordParam              string
	queryBuffer              string
	rowValueString           string
	statementPlaceholderList string
	statementValueList       string
	statement                string
	statementError           string
	singleRecord             string
	execResult               string
	execError                string
	affectedResult           string
	affectedError            string

	recordIndex string
}

// newCreateableGenerator returns a reader that will generate a record store's creation api.
func newCreateableGenerator(record marlowRecord) io.Reader {
	pr, pw := io.Pipe()
	methodName := fmt.Sprintf("Create%s", inflector.Pluralize(record.name()))

	symbols := createableSymbolList{
		recordParam:              "_records",
		queryBuffer:              "_query",
		rowValueString:           "_placeholders",
		statementPlaceholderList: "_placeholderList",
		statementValueList:       "_valueList",
		statement:                "_statement",
		statementError:           "_e",
		singleRecord:             "_record",
		execResult:               "_result",
		execError:                "_execError",
		affectedResult:           "_affectedResult",
		affectedError:            "_affectedError",
		recordIndex:              "_",
	}

	params := []writing.FuncParam{
		{Symbol: symbols.recordParam, Type: fmt.Sprintf("...%s", record.name())},
	}

	returns := []string{
		"int64",
		"error",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)

		gosrc.Comment("[marlow] createable")

		if record.dialect() == "postgres" {
			symbols.recordIndex = "_recordIndex"
		}

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			gosrc.WithIf("len(%s) == 0", func(url.Values) error {
				gosrc.Println("return 0, nil")
				return nil
			}, symbols.recordParam)

			columnList := make([]string, 0, len(record.fields))
			placeholders := make([]string, 0, len(record.fields))
			fieldLookup := make(map[string]string, len(record.fields))
			index := 1

			for field, c := range record.fields {
				columnName := c.Get(constants.ColumnConfigOption)

				if c.Get(constants.ColumnAutoIncrementFlag) != "" {
					continue
				}

				columnList = append(columnList, columnName)
				placeholder := "?"

				if record.dialect() == "postgres" {
					fmtStr := "fmt.Sprintf(\"$%%d\", (%s*(%d-1))+%d)"
					placeholder = fmt.Sprintf(fmtStr, symbols.recordIndex, len(record.fields), index)
				}

				placeholders = append(placeholders, placeholder)
				fieldLookup[columnName] = field
				index++
			}

			sort.Strings(columnList)

			gosrc.Println("%s := make([]string, 0, len(%s))", symbols.statementPlaceholderList, symbols.recordParam)
			gosrc.Println("%s := make([]interface{}, 0, len(%s))", symbols.statementValueList, symbols.recordParam)

			gosrc.WithIter("%s, %s := range %s", func(url.Values) error {
				if record.dialect() == "postgres" {
					gosrc.Println("%s := []string{%s}", symbols.rowValueString, strings.Join(placeholders, ", "))
				} else {
					gosrc.Println("%s := %s", symbols.rowValueString, writing.StringSliceLiteral(placeholders))
				}

				fieldReferences := make([]string, 0, len(columnList))

				for _, columnName := range columnList {
					field := fieldLookup[columnName]
					fieldReferences = append(fieldReferences, fmt.Sprintf("%s.%s", symbols.singleRecord, field))
				}

				gosrc.Println(
					"%s = append(%s, %s)",
					symbols.statementValueList,
					symbols.statementValueList,
					strings.Join(fieldReferences, ","),
				)

				gosrc.Println(
					"%s = append(%s, fmt.Sprintf(\"(%%s)\", strings.Join(%s, \",\")))",
					symbols.statementPlaceholderList,
					symbols.statementPlaceholderList,
					symbols.rowValueString,
				)
				return nil
			}, symbols.recordIndex, symbols.singleRecord, symbols.recordParam)

			gosrc.Println("%s := new(bytes.Buffer)", symbols.queryBuffer)

			insertStatement := fmt.Sprintf("INSERT INTO %s (%s) VALUES %%s;", record.table(), strings.Join(columnList, ","))

			if record.dialect() == "postgres" {
				template := "INSERT INTO %s (%s) VALUES %%s RETURNING %s;"
				columns := strings.Join(columnList, ",")
				primary := record.primaryKeyColumn()
				insertStatement = fmt.Sprintf(template, record.table(), columns, primary)
			}

			gosrc.Println(
				"fmt.Fprintf(%s, \"%s\", strings.Join(%s, \", \"))\n",
				symbols.queryBuffer,
				insertStatement,
				symbols.statementPlaceholderList,
			)

			gosrc.Println(
				"%s, %s := %s.Prepare(%s.String())",
				symbols.statement,
				symbols.statementError,
				scope.Get("receiver"),
				symbols.queryBuffer,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.statementError)
				return nil
			}, symbols.statementError)

			gosrc.Println("defer %s.Close()\n", symbols.statement)

			execution := "%s, %s := %s.Exec(%s...)"

			if record.dialect() == "postgres" {
				execution = "%s, %s := %s.Query(%s...)"
			}

			gosrc.Println(execution, symbols.execResult, symbols.execError, symbols.statement, symbols.statementValueList)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.execError)
				return nil
			}, symbols.execError)

			if record.dialect() != "postgres" {
				gosrc.Println("%s, %s := %s.LastInsertId()", symbols.affectedResult, symbols.affectedError, symbols.execResult)
				gosrc.Println("return %s, %s", symbols.affectedResult, symbols.affectedError)
				return nil
			}

			gosrc.Println("var %s int64", symbols.affectedResult)

			// Close the rows
			gosrc.Println("defer %s.Close()\n", symbols.execResult)

			// Iterate over rows scanning into result
			gosrc.WithIter("%s.Next()", func(url.Values) error {
				gosrc.WithIf("%s := %s.Scan(&%s); %s != nil", func(url.Values) error {
					gosrc.Println("return -1, %s", symbols.affectedError)
					return nil
				}, symbols.affectedError, symbols.execResult, symbols.affectedResult, symbols.affectedError)

				return nil
			}, symbols.execResult)

			gosrc.Println("return %s, nil", symbols.affectedResult)
			return nil
		})

		if e == nil {
			record.registerImports("fmt", "bytes", "strings")
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
