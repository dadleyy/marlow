package marlow

import "io"
import "fmt"
import "strings"
import "net/url"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type finderSymbols struct {
	blueprint       string
	results         string
	rowItem         string
	queryString     string
	statementResult string
	statementError  string
	queryResult     string
	queryError      string
	recordSlice     string
	limit           string
	offset          string
}

func finder(record marlowRecord) io.Reader {
	methodName := fmt.Sprintf("%s%s",
		record.config.Get(constants.StoreFindMethodPrefixConfigOption),
		inflector.Pluralize(record.name()),
	)
	pr, pw := io.Pipe()

	if len(record.fields) == 0 {
		pw.CloseWithError(nil)
		return pr
	}

	blueprintName := record.config.Get(constants.BlueprintNameConfigOption)

	symbols := finderSymbols{
		blueprint:       "_blueprint",
		results:         "_results",
		rowItem:         "_row",
		statementResult: "_statement",
		statementError:  "_se",
		queryString:     "_queryString",
		queryResult:     "_queryResult",
		queryError:      "_qe",
		limit:           "_limit",
		offset:          "_offset",
		recordSlice:     fmt.Sprintf("[]*%s", record.name()),
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)
		gosrc.Comment("[marlow feature]: finder on table[%s]", record.table())

		params := []writing.FuncParam{
			{Symbol: symbols.blueprint, Type: fmt.Sprintf("*%s", blueprintName)},
		}

		returns := []string{symbols.recordSlice, "error"}

		fieldList := record.fieldList(nil)
		defaultLimit := record.config.Get(constants.DefaultLimitConfigOption)

		if defaultLimit == "" {
			pw.CloseWithError(fmt.Errorf("invalid defaultLimit for record %s", record.name()))
			return
		}

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			// Prepare the array that will be returned.
			gosrc.Println("%s := make(%s, 0)\n", symbols.results, symbols.recordSlice)
			defer gosrc.Returns(symbols.results, writing.Nil)

			columns := make([]string, len(fieldList))

			for i, n := range fieldList {
				columns[i] = n.column
			}

			// Prepare the sql statement that will be sent to the DB.
			gosrc.Println(
				"%s := bytes.NewBufferString(\"SELECT %s FROM %s\")",
				symbols.queryString,
				strings.Join(columns, ","),
				record.table(),
			)

			// Write our where clauses
			e := gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Println("fmt.Fprintf(%s, \" %%s\", %s)", symbols.queryString, symbols.blueprint)
			}, symbols.blueprint)

			// Write the limit determining code.
			limitCondition := fmt.Sprintf("%s != nil && %s.Limit >= 1", symbols.blueprint, symbols.blueprint)
			gosrc.Println("%s := %s", symbols.limit, defaultLimit)

			e = gosrc.WithIf(limitCondition, func(url.Values) error {
				return gosrc.Println("%s = %s.Limit", symbols.limit, symbols.blueprint)
			})

			if e != nil {
				return e
			}

			// Write the offset determining code.
			offsetCondition := fmt.Sprintf("%s != nil && %s.Offset >= 1", symbols.blueprint, symbols.blueprint)
			gosrc.Println("%s := 0", symbols.offset)

			e = gosrc.WithIf(offsetCondition, func(url.Values) error {
				return gosrc.Println("%s = %s.Offset", symbols.offset, symbols.blueprint)
			})

			if e != nil {
				return e
			}

			// Write out the limit & offset query write.
			gosrc.Println(
				"fmt.Fprintf(%s, \" LIMIT %%d OFFSET %%d\", %s, %s)",
				symbols.queryString,
				symbols.limit,
				symbols.offset,
			)

			// Log out the query.
			gosrc.Println(
				"fmt.Fprintf(%s.%s, \"%%s\\n\", %s)",
				scope.Get("receiver"),
				constants.StoreLoggerField,
				symbols.queryString,
			)

			// Write the query execution statement.
			gosrc.Println(
				"%s, %s := %s.Prepare(%s.String())",
				symbols.statementResult,
				symbols.statementError,
				scope.Get("receiver"),
				symbols.queryString,
			)

			// Query has been executed, write out error handler
			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns(writing.Nil, symbols.statementError)
			}, symbols.statementError)

			// Write out result close deferred statement.
			gosrc.Println("defer %s.Close()", symbols.statementResult)

			gosrc.Println(
				"%s, %s := %s.Query(%s.Values()...)",
				symbols.queryResult,
				symbols.queryError,
				symbols.statementResult,
				symbols.blueprint,
			)

			// Check to see if the two results had an error
			gosrc.WithIf("%s != nil ", func(url.Values) error {
				return gosrc.Returns(writing.Nil, symbols.queryError)
			}, symbols.queryError)

			return gosrc.WithIter("%s.Next()", func(url.Values) error {
				gosrc.Println("var %s %s", symbols.rowItem, record.name())
				references := make([]string, 0, len(record.fields))

				for _, f := range fieldList {
					references = append(references, fmt.Sprintf("&%s.%s", symbols.rowItem, f.name))
				}

				scans := strings.Join(references, ",")
				condition := fmt.Sprintf("e := %s.Scan(%s); e != nil", symbols.queryResult, scans)

				gosrc.WithIf(condition, func(url.Values) error {
					gosrc.Println("return nil, e")
					return nil
				})

				gosrc.Println("%s = append(%s, &%s)", symbols.results, symbols.results, symbols.rowItem)
				return nil
			}, symbols.queryResult)
		})

		if e != nil {
			pw.CloseWithError(e)
			return
		}

		record.registerStoreMethod(writing.FuncDecl{
			Name:    methodName,
			Params:  params,
			Returns: returns,
		})
		record.registerImports("fmt", "bytes", "strings")

		pw.Close()
	}()

	return pr
}

type counterSymbols struct {
	countMethodName    string
	blueprintParamName string
	StatementQuery     string
	statementResult    string
	statementError     string
	queryResult        string
	queryError         string
	ScanResult         string
	scanError          string
}

func counter(record marlowRecord) io.Reader {
	pr, pw := io.Pipe()
	methodPrefix := record.config.Get(constants.StoreCountMethodPrefixConfigOption)

	if len(record.fields) == 0 {
		pw.CloseWithError(nil)
		return pr
	}

	symbols := counterSymbols{
		countMethodName:    fmt.Sprintf("%s%s", methodPrefix, inflector.Pluralize(record.name())),
		blueprintParamName: "_blueprint",
		StatementQuery:     "_raw",
		statementError:     "_statementError",
		statementResult:    "_statement",
		queryResult:        "_queryResult",
		queryError:         "_queryError",
		ScanResult:         "_scanResult",
		scanError:          "_scanError",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)
		gosrc.Comment("[marlow feature]: counter on table[%s]", record.table())

		params := []writing.FuncParam{
			{Symbol: symbols.blueprintParamName, Type: fmt.Sprintf("*%s", record.blueprint())},
		}

		returns := []string{
			"int",
			"error",
		}

		e := gosrc.WithMethod(symbols.countMethodName, record.store(), params, returns, func(scope url.Values) error {
			receiver := scope.Get("receiver")
			gosrc.WithIf("%s == nil", func(url.Values) error {
				return gosrc.Println("%s = &%s{}", params[0].Symbol, record.blueprint())
			}, symbols.blueprintParamName)

			gosrc.Println(
				"%s := fmt.Sprintf(\"SELECT COUNT(*) FROM %s %%s;\", %s)",
				symbols.StatementQuery,
				record.table(),
				symbols.blueprintParamName,
			)

			gosrc.Println(
				"%s, %s := %s.Prepare(%s)",
				symbols.statementResult,
				symbols.statementError,
				receiver,
				symbols.StatementQuery,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns("-1", symbols.statementError)
			}, symbols.statementError)

			gosrc.Println("defer %s.Close()", symbols.statementResult)

			gosrc.Println(
				"%s, %s := %s.Query(%s.Values()...)",
				symbols.queryResult,
				symbols.queryError,
				symbols.statementResult,
				symbols.blueprintParamName,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns("-1", symbols.queryError)
			}, symbols.queryError)

			gosrc.Println("defer %s.Close()", symbols.queryResult)

			gosrc.WithIf("%s.Next() != true", func(url.Values) error {
				return gosrc.Returns("-1", "fmt.Errorf(\"invalid-scan\")")
			}, symbols.queryResult)

			gosrc.Println("var %s int", symbols.ScanResult)
			gosrc.Println("%s := %s.Scan(&%s)", symbols.scanError, symbols.queryResult, symbols.ScanResult)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns("-1", symbols.scanError)
			}, symbols.scanError)

			return gosrc.Returns(symbols.ScanResult, writing.Nil)
		})

		if e == nil {
			record.registerImports("fmt")
			record.registerStoreMethod(writing.FuncDecl{
				Name:    symbols.countMethodName,
				Params:  params,
				Returns: returns,
			})
		}

		pw.CloseWithError(e)
	}()

	return pr
}

type selectorSymbols struct {
	returnSlice     string
	queryResult     string
	queryError      string
	queryString     string
	statementResult string
	statementError  string
	blueprintParam  string
	rowItem         string
	scanError       string
	limit           string
	offset          string
}

func selector(record marlowRecord, fieldName string, fieldConfig url.Values) io.Reader {
	pr, pw := io.Pipe()
	methodName := fmt.Sprintf("Select%s", inflector.Pluralize(fieldName))
	columnName := fieldConfig.Get(constants.ColumnConfigOption)

	returnItemType := fieldConfig.Get("type")
	returnArrayType := fmt.Sprintf("[]%s", returnItemType)

	returns := []string{
		returnArrayType,
		"error",
	}

	symbols := selectorSymbols{
		returnSlice:     "_results",
		queryString:     "_queryString",
		queryResult:     "_queryResult",
		queryError:      "_qe",
		statementResult: "_statement",
		statementError:  "_se",
		scanError:       "_re",
		blueprintParam:  "_blueprint",
		rowItem:         "_row",
		limit:           "_limit",
		offset:          "_offset",
	}

	params := []writing.FuncParam{
		{Type: fmt.Sprintf("*%s", record.blueprint()), Symbol: symbols.blueprintParam},
	}

	columnReference := fmt.Sprintf("%s.%s", record.table(), columnName)

	go func() {
		gosrc := writing.NewGoWriter(pw)

		gosrc.Comment("[marlow] field selector for %s (%s) [print: %s]", fieldName, methodName, record.blueprint())

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			gosrc.Println("%s := make(%s, 0)", symbols.returnSlice, returnArrayType)

			gosrc.Println(
				"%s := bytes.NewBufferString(\"SELECT %s FROM %s\")",
				symbols.queryString,
				columnReference,
				record.table(),
			)

			// Write our where clauses
			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Println("fmt.Fprintf(%s, \" %%s\", %s)", symbols.queryString, symbols.blueprintParam)
			}, symbols.blueprintParam)

			defaultLimit := record.config.Get(constants.DefaultLimitConfigOption)

			gosrc.Println("%s, %s := %s, 0", symbols.limit, symbols.offset, defaultLimit)

			gosrc.WithIf("%s != nil && %s.Offset > 0", func(url.Values) error {
				return gosrc.Println("%s = %s.Offset", symbols.offset, symbols.blueprintParam)
			}, symbols.blueprintParam, symbols.blueprintParam)

			gosrc.WithIf("%s != nil && %s.Limit > 0", func(url.Values) error {
				return gosrc.Println("%s = %s.Limit", symbols.limit, symbols.blueprintParam)
			}, symbols.blueprintParam, symbols.blueprintParam)

			rangeString := "\" LIMIT %d OFFSET %d\""
			gosrc.Println("fmt.Fprintf(%s, %s, %s, %s)", symbols.queryString, rangeString, symbols.limit, symbols.offset)

			// Log out the query.
			gosrc.Println(
				"fmt.Fprintf(%s.%s, \"%%s %%s\\n\", %s, %s.Values())",
				scope.Get("receiver"),
				constants.StoreLoggerField,
				symbols.queryString,
				symbols.blueprintParam,
			)

			// Write the query execution statement.
			gosrc.Println(
				"%s, %s := %s.Prepare(%s.String())",
				symbols.statementResult,
				symbols.statementError,
				scope.Get("receiver"),
				symbols.queryString,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns(writing.Nil, symbols.statementError)
			}, symbols.statementError)

			// Write out result close deferred statement.
			gosrc.Println("defer %s.Close()", symbols.statementResult)

			gosrc.Println(
				"%s, %s := %s.Query(%s.Values()...)",
				symbols.queryResult,
				symbols.queryError,
				symbols.statementResult,
				symbols.blueprintParam,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns(writing.Nil, symbols.queryError)
			}, symbols.queryError)

			// Write out result close deferred statement.
			gosrc.Println("defer %s.Close()", symbols.queryResult)

			e := gosrc.WithIter("%s.Next()", func(url.Values) error {
				gosrc.Println("var %s %s", symbols.rowItem, returnItemType)
				condition := fmt.Sprintf(
					"%s := %s.Scan(&%s); %s != nil",
					symbols.scanError,
					symbols.queryResult,
					symbols.rowItem,
					symbols.scanError,
				)

				gosrc.WithIf(condition, func(url.Values) error {
					return gosrc.Returns(writing.Nil, symbols.scanError)
				})

				return gosrc.Println("%s = append(%s, %s)", symbols.returnSlice, symbols.returnSlice, symbols.rowItem)
			}, symbols.queryResult)

			if e != nil {
				return e
			}

			record.registerStoreMethod(writing.FuncDecl{
				Name:    methodName,
				Params:  params,
				Returns: returns,
			})
			gosrc.Println("return %s, nil", symbols.returnSlice)
			return nil
		})

		pw.CloseWithError(e)
	}()

	return pr
}

// newQueryableGenerator is responsible for returning a reader that will generate lookup functions for a given record.
func newQueryableGenerator(record marlowRecord) io.Reader {
	pr, pw := io.Pipe()

	if len(record.table()) == 0 || len(record.name()) == 0 || len(record.store()) == 0 {
		pw.CloseWithError(fmt.Errorf("invalid record config"))
		return pr
	}

	features := []io.Reader{
		finder(record),
		counter(record),
	}

	for name, config := range record.fields {
		s := selector(record, name, config)
		features = append(features, s)
	}

	go func() {
		_, e := io.Copy(pw, io.MultiReader(features...))
		pw.CloseWithError(e)
	}()

	return pr
}
