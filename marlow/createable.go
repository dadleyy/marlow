package marlow

import "io"
import "fmt"
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

	if record.dialect() == "postgres" && record.primaryKeyColumn() == "" {
		pw.CloseWithError(fmt.Errorf("postgres records are required to have a primaryKey defined"))
		return pr
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)

		gosrc.Comment("[marlow] createable")

		if record.dialect() == "postgres" {
			symbols.recordIndex = "_recordIndex"
		}

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			logwriter := logWriter{output: gosrc, receiver: scope.Get("receiver")}

			gosrc.WithIf("len(%s) == 0", func(url.Values) error {
				return gosrc.Returns("0", writing.Nil)
			}, symbols.recordParam)

			columns := make([]string, 0, len(record.fields))
			placeholders := make([]string, 0, len(record.fields))
			index := 1

			// Skip fields that have the `autoIncrement` directive or those that are being managed by the library.
			fields := record.fieldList(func(config url.Values) bool {
				deletion := record.deletionField()

				if deletion != nil && deletion.Get("FieldName") == config.Get("FieldName") {
					return false
				}

				return config.Get(constants.ColumnAutoIncrementFlag) == ""
			})

			for _, field := range fields {
				placeholder := "?"

				if record.dialect() == "postgres" {
					fmtStr := "fmt.Sprintf(\"$%%d\", (%s*%d)+%d)"
					placeholder = fmt.Sprintf(fmtStr, symbols.recordIndex, len(fields), index)
				}

				columns = append(columns, strings.Split(field.column, ".")[1])
				placeholders = append(placeholders, placeholder)
				index++
			}

			gosrc.Println("%s := make([]string, 0, len(%s))", symbols.statementPlaceholderList, symbols.recordParam)
			gosrc.Println("%s := make([]interface{}, 0, len(%s))", symbols.statementValueList, symbols.recordParam)

			gosrc.WithIter("%s, %s := range %s", func(url.Values) error {
				if record.dialect() == "postgres" {
					gosrc.Println("%s := []string{%s}", symbols.rowValueString, strings.Join(placeholders, ", "))
				} else {
					gosrc.Println("%s := %s", symbols.rowValueString, writing.StringSliceLiteral(placeholders))
				}

				fieldReferences := make([]string, 0, len(placeholders))

				for _, field := range fields {
					config := record.fields[field.name]

					if config.Get(constants.ColumnAutoIncrementFlag) != "" {
						continue
					}

					fieldReferences = append(fieldReferences, fmt.Sprintf("%s.%s", symbols.singleRecord, field.name))
				}

				gosrc.Println(
					"%s = append(%s, %s)",
					symbols.statementValueList,
					symbols.statementValueList,
					strings.Join(fieldReferences, ","),
				)

				return gosrc.Println(
					"%s = append(%s, fmt.Sprintf(\"(%%s)\", strings.Join(%s, \",\")))",
					symbols.statementPlaceholderList,
					symbols.statementPlaceholderList,
					symbols.rowValueString,
				)
			}, symbols.recordIndex, symbols.singleRecord, symbols.recordParam)

			gosrc.Println("%s := new(bytes.Buffer)", symbols.queryBuffer)

			insertStatement := fmt.Sprintf("INSERT INTO %s (%s) VALUES %%s;", record.table(), strings.Join(columns, ","))

			if record.dialect() == "postgres" {
				template := "INSERT INTO %s (%s) VALUES %%s RETURNING %s;"
				primary := record.primaryKeyColumn()
				insertStatement = fmt.Sprintf(template, record.table(), strings.Join(columns, ","), primary)
			}

			gosrc.Println(
				"fmt.Fprintf(%s, \"%s\", strings.Join(%s, \", \"))\n",
				symbols.queryBuffer,
				insertStatement,
				symbols.statementPlaceholderList,
			)

			logwriter.AddLog(symbols.queryBuffer, symbols.statementValueList)

			gosrc.Println(
				"%s, %s := %s.Prepare(%s.String())",
				symbols.statement,
				symbols.statementError,
				scope.Get("receiver"),
				symbols.queryBuffer,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns("-1", symbols.statementError)
			}, symbols.statementError)

			gosrc.Println("defer %s.Close()\n", symbols.statement)

			execution := "%s, %s := %s.Exec(%s...)"

			if record.dialect() == "postgres" {
				execution = "%s, %s := %s.Query(%s...)"
			}

			gosrc.Println(execution, symbols.execResult, symbols.execError, symbols.statement, symbols.statementValueList)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns("-1", symbols.execError)
			}, symbols.execError)

			if record.dialect() != "postgres" {
				gosrc.Println("%s, %s := %s.LastInsertId()", symbols.affectedResult, symbols.affectedError, symbols.execResult)
				return gosrc.Returns(symbols.affectedResult, symbols.affectedError)
			}

			gosrc.Println("var %s int64", symbols.affectedResult)

			// Close the rows
			gosrc.Println("defer %s.Close()\n", symbols.execResult)

			// Iterate over rows scanning into result
			gosrc.WithIter("%s.Next()", func(url.Values) error {
				gosrc.WithIf("%s := %s.Scan(&%s); %s != nil", func(url.Values) error {
					return gosrc.Returns("-1", symbols.affectedError)
				}, symbols.affectedError, symbols.execResult, symbols.affectedResult, symbols.affectedError)

				return nil
			}, symbols.execResult)

			return gosrc.Returns(symbols.affectedResult, writing.Nil)
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
