package features

import "io"
import "fmt"
import "sort"
import "strings"
import "net/url"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

func finder(record url.Values, fields map[string]url.Values, imports chan<- string) io.Reader {
	table := record.Get(constants.TableNameConfigOption)
	recordName := record.Get(constants.RecordNameConfigOption)
	store := record.Get(constants.StoreNameConfigOption)
	pr, pw := io.Pipe()

	if len(fields) == 0 {
		pw.CloseWithError(nil)
		return pr
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)
		gosrc.Comment("[marlow feature]: counter on table[%s]", table)

		bp := blueprint{
			record: record,
			fields: fields,
		}

		symbols := map[string]string{
			"blueprint":         "_query",
			"RESULTS":           "_results",
			"FULL_QUERY_BUFFER": "_fullQuery",
			"ROW_RESULTS":       "_rowResults",
			"ROW_ITEM":          "_rowItem",
			"LIMIT":             "_limit",
			"OFFSET":            "_offset",
			"RECORD_SLICE":      fmt.Sprintf("[]*%s", recordName),

			"FUNC_NAME": fmt.Sprintf("%s%s",
				record.Get(constants.StoreFindMethodPrefixConfigOption),
				inflector.Pluralize(recordName),
			),
		}

		params := []writing.FuncParam{
			{Symbol: symbols["blueprint"], Type: fmt.Sprintf("*%s", bp.Name())},
		}

		returns := []string{symbols["RECORD_SLICE"], "error"}

		fieldList := make([]string, 0, len(fields))

		for name, config := range fields {
			colName := config.Get(constants.ColumnConfigOption)

			if colName == "" {
				colName = strings.ToLower(name)
			}

			expanded := fmt.Sprintf("%s.%s", table, colName)
			fieldList = append(fieldList, expanded)
		}

		defaultLimit := record.Get(constants.DefaultLimitConfigOption)

		if defaultLimit == "" {
			pw.CloseWithError(fmt.Errorf("invalid defaultLimit for record %s", recordName))
			return
		}

		sort.Strings(fieldList)

		e := gosrc.WithMethod(symbols["FUNC_NAME"], store, params, returns, func(scope url.Values) error {
			// Prepare the array that will be returned.
			gosrc.Println("%s := make(%s, 0)\n", symbols["RESULTS"], symbols["RECORD_SLICE"])
			defer gosrc.Println("return %s, nil", symbols["RESULTS"])

			// Prepare the sql statement that will be sent to the DB.
			gosrc.Println(
				"%s := bytes.NewBufferString(\"SELECT %s FROM %s\")",
				symbols["FULL_QUERY_BUFFER"],
				strings.Join(fieldList, ","),
				table,
			)

			// Write our where clauses
			e := gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("fmt.Fprintf(%s, \" %%s\", %s)", symbols["FULL_QUERY_BUFFER"], symbols["blueprint"])
				return nil
			}, symbols["blueprint"])

			// Write the limit determining code.
			limitCondition := fmt.Sprintf("%s != nil && %s.Limit >= 1", symbols["blueprint"], symbols["blueprint"])
			gosrc.Println("%s := %s", symbols["LIMIT"], defaultLimit)
			e = gosrc.WithIf(limitCondition, func(url.Values) error {
				gosrc.Println("%s = %s.Limit", symbols["LIMIT"], symbols["blueprint"])
				return nil
			})

			if e != nil {
				return e
			}

			// Write the offset determining code.
			offsetCondition := fmt.Sprintf("%s != nil && %s.Offset >= 1", symbols["blueprint"], symbols["blueprint"])
			gosrc.Println("%s := 0", symbols["OFFSET"])

			e = gosrc.WithIf(offsetCondition, func(url.Values) error {
				gosrc.Println("%s = %s.Offset", symbols["OFFSET"], symbols["blueprint"])
				return nil
			})

			if e != nil {
				return e
			}

			// Write out the limit & offset query write.
			gosrc.Println(
				"fmt.Fprintf(%s, \" LIMIT %%d OFFSET %%d\", %s, %s)",
				symbols["FULL_QUERY_BUFFER"],
				symbols["LIMIT"],
				symbols["OFFSET"],
			)

			// Write the query execution statement.
			gosrc.Println(
				"%s, e := %s.q(%s.String())",
				symbols["ROW_RESULTS"],
				scope.Get("receiver"),
				symbols["FULL_QUERY_BUFFER"],
			)

			// Query has been executed, write out error handler
			e = gosrc.WithIf("e != nil", func(url.Values) error {
				gosrc.Println("return nil, e")
				return nil
			})

			if e != nil {
				return e
			}

			// Write out result close deferred statement.
			gosrc.Println("defer %s.Close()", symbols["ROW_RESULTS"])

			// Check to see if the two results had an error
			gosrc.WithIf("e := %s.Err(); e != nil", func(url.Values) error {
				gosrc.Println("return nil, e")
				return nil
			}, symbols["ROW_RESULTS"])

			return gosrc.WithIter("%s.Next()", func(url.Values) error {
				gosrc.Println("var %s %s", symbols["ROW_ITEM"], recordName)
				references := make([]string, 0, len(fields))

				for name := range fields {
					references = append(references, fmt.Sprintf("&%s.%s", symbols["ROW_ITEM"], name))
				}

				sort.Strings(references)

				scans := strings.Join(references, ",")
				condition := fmt.Sprintf("e := %s.Scan(%s); e != nil", symbols["ROW_RESULTS"], scans)

				gosrc.WithIf(condition, func(url.Values) error {
					gosrc.Println("return nil, e")
					return nil
				})

				gosrc.Println("%s = append(%s, &%s)", symbols["RESULTS"], symbols["RESULTS"], symbols["ROW_ITEM"])
				return nil
			}, symbols["ROW_RESULTS"])
		})

		if e != nil {
			pw.CloseWithError(e)
			return
		}

		imports <- "fmt"
		imports <- "bytes"
		imports <- "strings"

		pw.Close()
	}()

	return pr
}

func counter(record url.Values, fields map[string]url.Values, imports chan<- string) io.Reader {
	table := record.Get(constants.TableNameConfigOption)
	recordName := record.Get(constants.RecordNameConfigOption)
	store := record.Get(constants.StoreNameConfigOption)
	blueprint := blueprint{record: record, fields: fields}
	pr, pw := io.Pipe()

	if len(fields) == 0 {
		pw.CloseWithError(nil)
		return pr
	}

	symbols := map[string]string{
		"COUNT_METHOD_NAME": fmt.Sprintf("%s%s",
			record.Get(constants.StoreCountMethodPrefixConfigOption),
			inflector.Pluralize(recordName),
		),
		"BLUEPRINT_PARAM_NAME": "_blueprint",
		"SELECTION_RESULT":     "_result",
		"SELECTION_QUERY":      "_fullQuery",
		"SELECTION_ERROR":      "_selectError",
		"SCAN_RESULT":          "_countResult",
		"SCAN_ERROR":           "_countError",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)
		gosrc.Comment("[marlow feature]: counter on table[%s]", table)

		params := []writing.FuncParam{
			{Symbol: symbols["BLUEPRINT_PARAM_NAME"], Type: fmt.Sprintf("*%s", blueprint.Name())},
		}

		returns := []string{
			"int",
			"error",
		}

		e := gosrc.WithMethod(symbols["COUNT_METHOD_NAME"], store, params, returns, func(scope url.Values) error {
			receiver := scope.Get("receiver")
			gosrc.WithIf("%s == nil", func(url.Values) error {
				gosrc.Println("%s = &%s{}", params[0].Symbol, blueprint.Name())
				return nil
			}, params[0].Symbol)

			gosrc.Println(
				"%s := fmt.Sprintf(\"SELECT COUNT(*) FROM %s %%s;\", %s)",
				symbols["SELECTION_QUERY"],
				table,
				params[0].Symbol,
			)

			gosrc.Println(
				"%s, %s := %s.q(%s)",
				symbols["SELECTION_RESULT"],
				symbols["SELECTION_ERROR"],
				receiver,
				symbols["SELECTION_QUERY"],
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols["SELECTION_ERROR"])
				return nil
			}, symbols["SELECTION_ERROR"])

			gosrc.WithIf("%s.Next() != true", func(url.Values) error {
				gosrc.Println("return -1, fmt.Errorf(\"invalid-scan\")")
				return nil
			}, symbols["SELECTION_RESULT"])

			gosrc.Println("var %s int", symbols["SCAN_RESULT"])
			gosrc.Println("%s := %s.Scan(&%s)", symbols["SCAN_ERROR"], symbols["SELECTION_RESULT"], symbols["SCAN_RESULT"])

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols["SCAN_ERROR"])
				return nil
			}, symbols["SCAN_ERROR"])

			gosrc.Println("return %s, nil", symbols["SCAN_RESULT"])
			return nil
		})

		if e == nil {
			imports <- "fmt"
		}

		pw.CloseWithError(e)
	}()

	return pr
}

func selector(record url.Values, name string, config url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()
	blueprint := blueprint{record: record}
	methodName := fmt.Sprintf("Select%s", inflector.Pluralize(name))

	tableName := record.Get(constants.TableNameConfigOption)
	columnName := config.Get(constants.ColumnConfigOption)
	storeName := record.Get(constants.StoreNameConfigOption)

	returnItemType := config.Get("type")
	returnArrayType := fmt.Sprintf("[]%s", returnItemType)

	returns := []string{
		returnArrayType,
		"error",
	}

	symbols := map[string]string{
		"RETURN_ARRAY":    "_results",
		"QUERY_BUFFER":    "_query",
		"BLUEPRINT_PARAM": "_blueprint",
		"ROW_RESULTS":     "_rows",
		"ROW_ITEM":        "_row",
	}

	params := []writing.FuncParam{
		{Type: fmt.Sprintf("*%s", blueprint.Name()), Symbol: symbols["BLUEPRINT_PARAM"]},
	}

	columnReference := fmt.Sprintf("%s.%s", tableName, columnName)

	go func() {
		gosrc := writing.NewGoWriter(pw)

		gosrc.Comment("[marlow] field selector for %s (%s) [print: %s]", name, methodName, blueprint.Name())

		e := gosrc.WithMethod(methodName, storeName, params, returns, func(scope url.Values) error {
			gosrc.Println("%s := make(%s, 0)", symbols["RETURN_ARRAY"], returnArrayType)

			gosrc.Println(
				"%s := bytes.NewBufferString(\"SELECT %s FROM %s\")",
				symbols["QUERY_BUFFER"],
				columnReference,
				tableName,
			)

			// Write our where clauses
			e := gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("fmt.Fprintf(%s, \" %%s\", %s)", symbols["QUERY_BUFFER"], symbols["BLUEPRINT_PARAM"])
				return nil
			}, symbols["BLUEPRINT_PARAM"])

			if e != nil {
				return e
			}

			// Write the query execution statement.
			gosrc.Println(
				"%s, e := %s.q(%s.String())",
				symbols["ROW_RESULTS"],
				scope.Get("receiver"),
				symbols["QUERY_BUFFER"],
			)

			gosrc.WithIf("e != nil", func(url.Values) error {
				gosrc.Println("return nil, e")
				return nil
			})

			// Write out result close deferred statement.
			gosrc.Println("defer %s.Close()", symbols["ROW_RESULTS"])

			e = gosrc.WithIter("%s.Next()", func(url.Values) error {
				gosrc.Println("var %s %s", symbols["ROW_ITEM"], returnItemType)
				condition := fmt.Sprintf("e := %s.Scan(&%s); e != nil", symbols["ROW_RESULTS"], symbols["ROW_ITEM"])

				gosrc.WithIf(condition, func(url.Values) error {
					gosrc.Println("return nil, e")
					return nil
				})

				gosrc.Println("%s = append(%s, %s)", symbols["RETURN_ARRAY"], symbols["RETURN_ARRAY"], symbols["ROW_ITEM"])
				return nil
			}, symbols["ROW_RESULTS"])

			if e != nil {
				return e
			}

			gosrc.Println("return %s, nil", symbols["RETURN_ARRAY"])
			return nil
		})

		pw.CloseWithError(e)
	}()

	return pr
}

// NewQueryableGenerator is responsible for returning a reader that will generate lookup functions for a given record.
func NewQueryableGenerator(record url.Values, fields map[string]url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()

	table := record.Get(constants.TableNameConfigOption)
	recordName := record.Get(constants.RecordNameConfigOption)
	store := record.Get(constants.StoreNameConfigOption)

	if len(table) == 0 || len(recordName) == 0 || len(store) == 0 {
		pw.CloseWithError(fmt.Errorf("invalid record config"))
		return pr
	}

	features := []io.Reader{
		finder(record, fields, imports),
		counter(record, fields, imports),
	}

	for name, config := range fields {
		features = append(features, selector(record, name, config, imports))
	}

	go func() {
		_, e := io.Copy(pw, io.MultiReader(features...))
		pw.CloseWithError(e)
	}()

	return pr
}
