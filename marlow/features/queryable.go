package features

import "io"
import "fmt"
import "sort"
import "strings"
import "net/url"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type importsChannel chan<- string

func finder(writer io.Writer, record url.Values, fields map[string]url.Values, imports importsChannel) error {
	table := record.Get(constants.TableNameConfigOption)
	recordName := record.Get(constants.RecordNameConfigOption)
	store := record.Get(constants.StoreNameConfigOption)

	if table == "" || recordName == "" || store == "" {
		return fmt.Errorf("invalid-record")
	}

	if len(fields) >= 1 == false {
		return io.EOF
	}

	out := writing.NewGoWriter(writer)

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
		colName := config.Get("column")

		if colName == "" {
			colName = strings.ToLower(name)
		}

		expanded := fmt.Sprintf("%s.%s", table, colName)
		fieldList = append(fieldList, expanded)
	}

	defaultLimit := record.Get(constants.DefaultLimitConfigOption)

	if defaultLimit == "" {
		return fmt.Errorf("invalid defaultLimit for record %s", recordName)
	}

	sort.Strings(fieldList)

	e := out.WithMethod(symbols["FUNC_NAME"], store, params, returns, func(scope url.Values) error {
		// Prepare the array that will be returned.
		out.Println("%s := make(%s, 0)\n", symbols["RESULTS"], symbols["RECORD_SLICE"])
		defer out.Println("return %s, nil", symbols["RESULTS"])

		// Prepare the sql statement that will be sent to the DB.
		out.Println(
			"%s := bytes.NewBufferString(\"SELECT %s FROM %s\")",
			symbols["FULL_QUERY_BUFFER"],
			strings.Join(fieldList, ","),
			table,
		)

		// Write our where clauses
		e := out.WithIf("%s != nil", func(url.Values) error {
			out.Println("fmt.Fprintf(%s, \" %%s\", %s)", symbols["FULL_QUERY_BUFFER"], symbols["blueprint"])
			return nil
		}, symbols["blueprint"])

		// Write the limit determining code.
		limitCondition := fmt.Sprintf("%s != nil && %s.Limit >= 1", symbols["blueprint"], symbols["blueprint"])
		out.Println("%s := %s", symbols["LIMIT"], defaultLimit)
		e = out.WithIf(limitCondition, func(url.Values) error {
			out.Println("%s = %s.Limit", symbols["LIMIT"], symbols["blueprint"])
			return nil
		})

		if e != nil {
			return e
		}

		// Write the offset determining code.
		offsetCondition := fmt.Sprintf("%s != nil && %s.Offset >= 1", symbols["blueprint"], symbols["blueprint"])
		out.Println("%s := 0", symbols["OFFSET"])

		e = out.WithIf(offsetCondition, func(url.Values) error {
			out.Println("%s = %s.Offset", symbols["OFFSET"], symbols["blueprint"])
			return nil
		})

		if e != nil {
			return e
		}

		// Write out the limit & offset query write.
		out.Println(
			"fmt.Fprintf(%s, \" LIMIT %%d OFFSET %%d\", %s, %s)",
			symbols["FULL_QUERY_BUFFER"],
			symbols["LIMIT"],
			symbols["OFFSET"],
		)

		// Write the query execution statement.
		out.Println("%s, e := %s.q(%s.String())", symbols["ROW_RESULTS"], scope.Get("receiver"), symbols["FULL_QUERY_BUFFER"])

		// Query has been executed, write out error handler
		e = out.WithIf("e != nil", func(url.Values) error {
			out.Println("return nil, e")
			return nil
		})

		if e != nil {
			return e
		}

		// Write out result close deferred statement.
		out.Println("defer %s.Close()", symbols["ROW_RESULTS"])

		// Check to see if the two results had an error
		out.WithIf("e := %s.Err(); e != nil", func(url.Values) error {
			out.Println("return nil, e")
			return nil
		}, symbols["ROW_RESULTS"])

		return out.WithIter("%s.Next()", func(url.Values) error {
			out.Println("var %s %s", symbols["ROW_ITEM"], recordName)
			references := make([]string, 0, len(fields))

			for name := range fields {
				references = append(references, fmt.Sprintf("&%s.%s", symbols["ROW_ITEM"], name))
			}

			sort.Strings(references)

			scans := strings.Join(references, ",")
			condition := fmt.Sprintf("e := %s.Scan(%s); e != nil", symbols["ROW_RESULTS"], scans)

			out.WithIf(condition, func(url.Values) error {
				out.Println("return nil, e")
				return nil
			})

			out.Println("%s = append(%s, &%s)", symbols["RESULTS"], symbols["RESULTS"], symbols["ROW_ITEM"])
			return nil
		}, symbols["ROW_RESULTS"])
	})

	if e != nil {
		return e
	}

	imports <- "fmt"
	imports <- "bytes"
	imports <- "strings"
	return nil
}

func counter(writer io.Writer, record url.Values, fields map[string]url.Values, imports importsChannel) error {
	table := record.Get(constants.TableNameConfigOption)
	recordName := record.Get(constants.RecordNameConfigOption)
	store := record.Get(constants.StoreNameConfigOption)

	symbols := map[string]string{
		"COUNT_METHOD_NAME": fmt.Sprintf("%s%s",
			record.Get(constants.StoreCountMethodPrefixConfigOption),
			inflector.Pluralize(recordName),
		),
	}

	gosrc := writing.NewGoWriter(writer)

	gosrc.Comment("[marlow feature]: counter on table[%s]", table)

	return gosrc.WithMethod(symbols["COUNT_METHOD_NAME"], store, nil, nil, func(url.Values) error {
		gosrc.Comment("[marlow feature]: counter on table[%s]", table)
		return nil
	})
}

type queryableFeatureWriter func(io.Writer, url.Values, map[string]url.Values, importsChannel) error

// NewQueryableGenerator is responsible for returning a reader that will generate lookup functions for a given record.
func NewQueryableGenerator(record url.Values, fields map[string]url.Values, imports importsChannel) io.Reader {
	pr, pw := io.Pipe()

	features := []queryableFeatureWriter{
		finder,
		counter,
	}

	go func() {
		var e error

		for _, f := range features {
			e = f(pw, record, fields, imports)

			if e != nil {
				break
			}
		}

		pw.CloseWithError(e)
	}()

	return pr
}
