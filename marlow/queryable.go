package marlow

import "io"
import "fmt"
import "sort"
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

		fieldList := make([]string, 0, len(record.fields))

		for name, config := range record.fields {
			colName := config.Get(constants.ColumnConfigOption)

			if colName == "" {
				colName = strings.ToLower(name)
			}

			expanded := fmt.Sprintf("%s.%s", record.table(), colName)
			fieldList = append(fieldList, expanded)
		}

		defaultLimit := record.config.Get(constants.DefaultLimitConfigOption)

		if defaultLimit == "" {
			pw.CloseWithError(fmt.Errorf("invalid defaultLimit for record %s", record.name()))
			return
		}

		sort.Strings(fieldList)

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			// Prepare the array that will be returned.
			gosrc.Println("%s := make(%s, 0)\n", symbols.results, symbols.recordSlice)
			defer gosrc.Println("return %s, nil", symbols.results)

			// Prepare the sql statement that will be sent to the DB.
			gosrc.Println(
				"%s := bytes.NewBufferString(\"SELECT %s FROM %s\")",
				symbols.queryString,
				strings.Join(fieldList, ","),
				record.table(),
			)

			// Write our where clauses
			e := gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("fmt.Fprintf(%s, \" %%s\", %s)", symbols.queryString, symbols.blueprint)
				return nil
			}, symbols.blueprint)

			// Write the limit determining code.
			limitCondition := fmt.Sprintf("%s != nil && %s.Limit >= 1", symbols.blueprint, symbols.blueprint)
			gosrc.Println("%s := %s", symbols.limit, defaultLimit)

			e = gosrc.WithIf(limitCondition, func(url.Values) error {
				gosrc.Println("%s = %s.Limit", symbols.limit, symbols.blueprint)
				return nil
			})

			if e != nil {
				return e
			}

			// Write the offset determining code.
			offsetCondition := fmt.Sprintf("%s != nil && %s.Offset >= 1", symbols.blueprint, symbols.blueprint)
			gosrc.Println("%s := 0", symbols.offset)

			e = gosrc.WithIf(offsetCondition, func(url.Values) error {
				gosrc.Println("%s = %s.Offset", symbols.offset, symbols.blueprint)
				return nil
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
				gosrc.Println("return nil, %s", symbols.statementError)
				return nil
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
				gosrc.Println("return nil, %s", symbols.queryError)
				return nil
			}, symbols.queryError)

			return gosrc.WithIter("%s.Next()", func(url.Values) error {
				gosrc.Println("var %s %s", symbols.rowItem, record.name())
				references := make([]string, 0, len(record.fields))

				for name := range record.fields {
					references = append(references, fmt.Sprintf("&%s.%s", symbols.rowItem, name))
				}

				sort.Strings(references)

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

		record.registerImports("fmt", "bytes", "strings")

		pw.Close()
	}()

	return pr
}

type counterSymbols struct {
	CountMethodName    string
	BlueprintParamName string
	StatementQuery     string
	StatementResult    string
	StatementError     string
	QueryResult        string
	QueryError         string
	ScanResult         string
	ScanError          string
}

func counter(record marlowRecord) io.Reader {
	pr, pw := io.Pipe()
	methodPrefix := record.config.Get(constants.StoreCountMethodPrefixConfigOption)

	if len(record.fields) == 0 {
		pw.CloseWithError(nil)
		return pr
	}

	symbols := counterSymbols{
		CountMethodName:    fmt.Sprintf("%s%s", methodPrefix, inflector.Pluralize(record.name())),
		BlueprintParamName: "_blueprint",
		StatementQuery:     "_raw",
		StatementError:     "_statementError",
		StatementResult:    "_statement",
		QueryResult:        "_queryResult",
		QueryError:         "_queryError",
		ScanResult:         "_scanResult",
		ScanError:          "_scanError",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)
		gosrc.Comment("[marlow feature]: counter on table[%s]", record.table())

		params := []writing.FuncParam{
			{Symbol: symbols.BlueprintParamName, Type: fmt.Sprintf("*%s", record.blueprint())},
		}

		returns := []string{
			"int",
			"error",
		}

		e := gosrc.WithMethod(symbols.CountMethodName, record.store(), params, returns, func(scope url.Values) error {
			receiver := scope.Get("receiver")
			gosrc.WithIf("%s == nil", func(url.Values) error {
				gosrc.Println("%s = &%s{}", params[0].Symbol, record.blueprint())
				return nil
			}, symbols.BlueprintParamName)

			gosrc.Println(
				"%s := fmt.Sprintf(\"SELECT COUNT(*) FROM %s %%s;\", %s)",
				symbols.StatementQuery,
				record.table(),
				symbols.BlueprintParamName,
			)

			gosrc.Println(
				"%s, %s := %s.Prepare(%s)",
				symbols.StatementResult,
				symbols.StatementError,
				receiver,
				symbols.StatementQuery,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.StatementError)
				return nil
			}, symbols.StatementError)

			gosrc.Println("defer %s.Close()", symbols.StatementResult)

			gosrc.Println(
				"%s, %s := %s.Query(%s.Values()...)",
				symbols.QueryResult,
				symbols.QueryError,
				symbols.StatementResult,
				symbols.BlueprintParamName,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.QueryError)
				return nil
			}, symbols.QueryError)

			gosrc.Println("defer %s.Close()", symbols.QueryResult)

			gosrc.WithIf("%s.Next() != true", func(url.Values) error {
				gosrc.Println("return -1, fmt.Errorf(\"invalid-scan\")")
				return nil
			}, symbols.QueryResult)

			gosrc.Println("var %s int", symbols.ScanResult)
			gosrc.Println("%s := %s.Scan(&%s)", symbols.ScanError, symbols.QueryResult, symbols.ScanResult)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.ScanError)
				return nil
			}, symbols.ScanError)

			gosrc.Println("return %s, nil", symbols.ScanResult)
			return nil
		})

		if e == nil {
			record.registerImports("fmt")
		}

		pw.CloseWithError(e)
	}()

	return pr
}

type selectorSymbols struct {
	ReturnSlice     string
	QueryResult     string
	QueryError      string
	QueryString     string
	StatementResult string
	StatementError  string
	BlueprintParam  string
	RowItem         string
	ScanError       string
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
		ReturnSlice:     "_results",
		QueryString:     "_queryString",
		QueryResult:     "_queryResult",
		QueryError:      "_qe",
		StatementResult: "_statement",
		StatementError:  "_se",
		ScanError:       "_re",
		BlueprintParam:  "_blueprint",
		RowItem:         "_row",
	}

	params := []writing.FuncParam{
		{Type: fmt.Sprintf("*%s", record.blueprint()), Symbol: symbols.BlueprintParam},
	}

	columnReference := fmt.Sprintf("%s.%s", record.table(), columnName)

	go func() {
		gosrc := writing.NewGoWriter(pw)

		gosrc.Comment("[marlow] field selector for %s (%s) [print: %s]", fieldName, methodName, record.blueprint())

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			gosrc.Println("%s := make(%s, 0)", symbols.ReturnSlice, returnArrayType)

			gosrc.Println(
				"%s := bytes.NewBufferString(\"SELECT %s FROM %s\")",
				symbols.QueryString,
				columnReference,
				record.table(),
			)

			// Write our where clauses
			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("fmt.Fprintf(%s, \" %%s\", %s)", symbols.QueryString, symbols.BlueprintParam)
				return nil
			}, symbols.BlueprintParam)

			// Write the query execution statement.
			gosrc.Println(
				"%s, %s := %s.Prepare(%s.String())",
				symbols.StatementResult,
				symbols.StatementError,
				scope.Get("receiver"),
				symbols.QueryString,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return nil, %s", symbols.StatementError)
				return nil
			}, symbols.StatementError)

			// Write out result close deferred statement.
			gosrc.Println("defer %s.Close()", symbols.StatementResult)

			gosrc.Println(
				"%s, %s := %s.Query(%s.Values()...)",
				symbols.QueryResult,
				symbols.QueryError,
				symbols.StatementResult,
				symbols.BlueprintParam,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return nil, %s", symbols.QueryError)
				return nil
			}, symbols.QueryError)

			// Write out result close deferred statement.
			gosrc.Println("defer %s.Close()", symbols.QueryResult)

			e := gosrc.WithIter("%s.Next()", func(url.Values) error {
				gosrc.Println("var %s %s", symbols.RowItem, returnItemType)
				condition := fmt.Sprintf(
					"%s := %s.Scan(&%s); %s != nil",
					symbols.ScanError,
					symbols.QueryResult,
					symbols.RowItem,
					symbols.ScanError,
				)

				gosrc.WithIf(condition, func(url.Values) error {
					gosrc.Println("return nil, %s", symbols.ScanError)
					return nil
				})

				gosrc.Println("%s = append(%s, %s)", symbols.ReturnSlice, symbols.ReturnSlice, symbols.RowItem)
				return nil
			}, symbols.QueryResult)

			if e != nil {
				return e
			}

			gosrc.Println("return %s, nil", symbols.ReturnSlice)
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
