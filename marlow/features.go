package marlow

import "io"
import "log"
import "fmt"
import "sort"
import "net/url"
import "strings"
import "github.com/gedex/inflector"

func copyRecordFinder(destination io.Writer, source *tableSource) error {
	singular := inflector.Singularize(source.recordName)
	plural := inflector.Pluralize(source.recordName)
	queryStructName := fmt.Sprintf("%sQuery", singular)

	tableName := source.tableName()

	w := goWriter{Logger: log.New(destination, "", 0)}

	w.Println("// [marlow feature]: record finder")

	lookupFnName := fmt.Sprintf("Find%s", plural)

	w.withStruct(queryStructName, func(url.Values) error {
		for name, config := range source.queryableFields() {
			w.Printf("%s []%s", name, config.Get("type"))
		}

		w.Printf("Limit uint")
		w.Printf("Offset uint")

		return nil
	})

	lookupParams := []funcParam{
		{paramName: "q", typeName: fmt.Sprintf("*%s", queryStructName)},
	}

	lookupReturns := []string{fmt.Sprintf("[]*%s", source.recordName), "error"}
	fieldList := make([]string, 0, len(source.fields))

	for n, c := range source.fields {
		colName := c.Get("column")

		if colName == "" {
			n = strings.ToLower(n)
		}

		fieldList = append(fieldList, colName)
	}

	sort.Strings(fieldList)

	w.withMetod(lookupFnName, source.storeName(), lookupParams, lookupReturns, func(scope url.Values) error {
		w.Printf("query := fmt.Sprintf(\"SELECT %s FROM %s;\")", strings.Join(fieldList, ","), tableName)
		w.Printf("results, e := %s.q(query)", scope.Get("receiver"))
		w.Printf("out := make([]*%s, 0)", source.recordName)
		w.Println()

		w.withIf("e != nil", func(url.Values) error {
			w.Printf("return nil, e")
			return nil
		})

		w.Println("defer results.Close()")
		w.Println("")

		w.withIf("e := results.Err(); e != nil", func(url.Values) error {
			w.Printf("return nil, e")
			return nil
		})

		w.withIter("results.Next()", func(url.Values) error {
			w.Printf("var b %s", source.recordName)
			references := make([]string, 0, len(source.fields))

			for name := range source.fields {
				references = append(references, fmt.Sprintf("&b.%s", name))
			}

			sort.Strings(references)

			scans := strings.Join(references, ",")
			condition := fmt.Sprintf("e := results.Scan(%s); e != nil", scans)

			w.withIf(condition, func(url.Values) error {
				w.Printf("return nil, e")
				return nil
			})

			w.Println("out = append(out, &b)")
			return nil
		})

		w.Println("return out, nil")
		return nil
	})

	return nil
}

func recordFinderGenerator(source *tableSource) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		e := copyRecordFinder(pw, source)
		pw.CloseWithError(e)
	}()

	return pr
}
