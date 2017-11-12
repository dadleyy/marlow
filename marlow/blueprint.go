package marlow

import "io"
import "fmt"
import "sync"
import "strings"
import "net/url"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

func writeBlueprint(destination io.Writer, record marlowRecord) error {
	out := writing.NewGoWriter(destination)

	e := out.WithStruct(record.blueprint(), func(url.Values) error {
		for name, config := range record.fields {
			fieldType := config.Get("type")

			if fieldType == "" {
				return fmt.Errorf("bad field type for field name: %s", name)
			}

			// Support IN lookup on string fields.
			if fieldType == "int" {
				out.Println("%s%s []int", name, record.config.Get(constants.BlueprintRangeFieldSuffixConfigOption))
			}

			// Support LIKE lookup on string fields.
			if fieldType == "string" {
				out.Println("%s%s []string", name, record.config.Get(constants.BlueprintLikeFieldSuffixConfigOption))
			}

			if fieldImport := config.Get("import"); fieldImport != "" {
				record.registerImports(fieldImport)
			}

			out.Println("%s []%s", name, fieldType)
		}

		out.Println("Limit int")
		out.Println("Offset int")
		out.Println("OrderBy string")
		out.Println("OrderDirection string")

		return nil
	})

	if e != nil {
		return e
	}

	record.registerImports("fmt", "strings")

	var readers []io.Reader
	methodReceiver := make(chan string)
	var clauseMethods []string
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		for name := range methodReceiver {
			clauseMethods = append(clauseMethods, name)
		}
		wg.Done()
	}()

	for name, config := range record.fields {
		fieldGenerators := fieldMethods(record, name, config, methodReceiver)

		if len(fieldGenerators) == 0 {
			continue
		}

		readers = append(readers, fieldGenerators...)
	}

	if _, e := io.Copy(destination, io.MultiReader(readers...)); e != nil {
		return e
	}

	close(methodReceiver)
	wg.Wait()

	symbols := struct {
		ClauseSlice string
		ClauseItem  string
	}{"_clauses", "_item"}

	// With all of our fields having generated non-exported clause generation methods on our struct, we can create the
	// final 'String' method which iterates over all of these, calling them and adding the non-empty string clauses to
	// a list, which eventually is returned as a joined string.
	e = out.WithMethod("String", record.blueprint(), nil, []string{"string"}, func(scope url.Values) error {
		out.Println("%s := make([]string, 0, %d)", symbols.ClauseSlice, len(clauseMethods))

		for _, method := range clauseMethods {
			out.WithIf("%s, _ := %s.%s(); %s != \"\"", func(url.Values) error {
				out.Println("%s = append(%s, %s)", symbols.ClauseSlice, symbols.ClauseSlice, symbols.ClauseItem)
				return nil
			}, symbols.ClauseItem, scope.Get("receiver"), method, symbols.ClauseItem)
		}

		out.WithIf("len(%s) == 0", func(url.Values) error {
			out.Println("return \"\"")
			return nil
		}, symbols.ClauseSlice)

		out.Println("return \"WHERE \" + strings.Join(%s, \" AND \")", symbols.ClauseSlice)
		return nil
	})

	if e != nil {
		return e
	}

	return out.WithMethod("Values", record.blueprint(), nil, []string{"[]interface{}"}, func(scope url.Values) error {
		out.Println("%s := make([]interface{}, 0, %d)", symbols.ClauseSlice, len(clauseMethods))

		out.WithIf("%s == nil", func(url.Values) error {
			out.Println("return nil")
			return nil
		}, scope.Get("receiver"))

		for _, method := range clauseMethods {
			out.WithIf("_, %s := %s.%s(); %s != nil && len(%s) > 0", func(url.Values) error {
				out.Println("%s = append(%s, %s...)", symbols.ClauseSlice, symbols.ClauseSlice, symbols.ClauseItem)
				return nil
			}, symbols.ClauseItem, scope.Get("receiver"), method, symbols.ClauseItem, symbols.ClauseItem)
		}

		out.Println("return %s", symbols.ClauseSlice)
		return nil

	})
}

func fieldMethods(record marlowRecord, name string, config url.Values, methods chan<- string) []io.Reader {
	fieldType := config.Get("type")
	results := make([]io.Reader, 0, len(record.fields))

	if fieldType == "string" || fieldType == "int" {
		results = append(results, simpleTypeIn(record, name, config, methods))
	}

	if fieldType == "string" {
		results = append(results, stringMethods(record, name, config, methods))
	}

	if fieldType == "int" {
		results = append(results, numericalMethods(record, name, config, methods))
	}

	if fieldType == "sql.NullInt64" {
		results = append(results, nullableIntMethods(record, name, config, methods))
	}

	if len(results) == 0 {
		warning := fmt.Sprintf("// [marlow] field %s (%s) unable to generate clauses. unsupported type", name, fieldType)

		results = []io.Reader{strings.NewReader(warning)}
	}

	return results
}

func nullableIntMethods(record marlowRecord, fieldName string, config url.Values, methods chan<- string) io.Reader {
	pr, pw := io.Pipe()
	columnName := config.Get(constants.ColumnConfigOption)
	methodName := fmt.Sprintf("%sInString", columnName)

	symbols := struct {
		PlaceholderSlice string
		ValueSlice       string
		ValueItem        string
		JoinedValues     string
	}{"_placeholders", "_values", "_v", "_joined"}

	columnReference := fmt.Sprintf("%s.%s", record.table(), columnName)

	returns := []string{"string", "[]interface{}"}

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] nullable clause gen for \"%s\"", columnReference)

		e := writer.WithMethod(methodName, record.blueprint(), nil, returns, func(scope url.Values) error {
			fieldReference := fmt.Sprintf("%s.%s", scope.Get("receiver"), fieldName)

			// Add conditional check for length presence on lookup slice.
			writer.WithIf("%s == nil", func(url.Values) error {
				writer.Println("return \"\", nil")
				return nil
			}, fieldReference)

			// Add conditional check for length presence on lookup slice.
			writer.WithIf("len(%s) == 0", func(url.Values) error {
				writer.Println("return \"%s NOT NULL\", nil", columnReference)
				return nil
			}, fieldReference)

			writer.Println("%s := make([]string, 0, len(%s))", symbols.PlaceholderSlice, fieldReference)
			writer.Println("%s := make([]interface{}, 0, len(%s))", symbols.ValueSlice, fieldReference)

			writer.WithIter("_, %s := range %s", func(url.Values) error {
				writer.WithIf("%s.Valid == false", func(url.Values) error {
					writer.Println("return \"%s IS NULL\", nil", columnReference)
					return nil
				}, symbols.ValueItem)

				writer.Println("%s = append(%s, \"?\")", symbols.PlaceholderSlice, symbols.PlaceholderSlice)
				writer.Println("%s = append(%s, %s)", symbols.ValueSlice, symbols.ValueSlice, symbols.ValueItem)
				return nil
			}, symbols.ValueItem, fieldReference)

			writer.Println("%s := strings.Join(%s, \",\")", symbols.JoinedValues, symbols.PlaceholderSlice)
			writer.Println(
				"return fmt.Sprintf(\"%s.%s IN (%%s)\", %s), %s",
				record.table(),
				columnName,
				symbols.JoinedValues,
				symbols.ValueSlice,
			)
			return nil
		})

		if e == nil {
			methods <- methodName
		}

		pw.CloseWithError(e)
	}

	go write()

	return pr
}

func simpleTypeIn(record marlowRecord, fieldName string, fieldConfig url.Values, methods chan<- string) io.Reader {
	pr, pw := io.Pipe()
	columnName := fieldConfig.Get(constants.ColumnConfigOption)
	methodName := fmt.Sprintf("%sInString", columnName)
	columnReference := fmt.Sprintf("%s.%s", record.table(), columnName)

	symbols := struct {
		PlaceholderSlice string
		ValueSlice       string
		ValueItem        string
		JoinedValues     string
	}{"_placeholder", "_values", "_v", "_joined"}

	returns := []string{"string", "[]interface{}"}

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] type IN clause for \"%s\"", columnReference)

		e := writer.WithMethod(methodName, record.blueprint(), nil, returns, func(scope url.Values) error {
			fieldReference := fmt.Sprintf("%s.%s", scope.Get("receiver"), fieldName)

			// Add conditional check for length presence on lookup slice.
			writer.WithIf("len(%s) == 0", func(url.Values) error {
				writer.Println("return \"\", nil")
				return nil
			}, fieldReference)

			writer.Println("%s := make([]string, 0, len(%s))", symbols.PlaceholderSlice, fieldReference)
			writer.Println("%s := make([]interface{}, 0, len(%s))", symbols.ValueSlice, fieldReference)

			writer.WithIter("_, %s := range %s", func(url.Values) error {
				writer.Println("%s = append(%s, \"?\")", symbols.PlaceholderSlice, symbols.PlaceholderSlice)
				writer.Println("%s = append(%s, %s)", symbols.ValueSlice, symbols.ValueSlice, symbols.ValueItem)
				return nil
			}, symbols.ValueItem, fieldReference)

			writer.Println("%s := strings.Join(%s, \",\")", symbols.JoinedValues, symbols.PlaceholderSlice)
			writer.Println(
				"return fmt.Sprintf(\"%s IN (%%s)\", %s), %s",
				columnReference,
				symbols.JoinedValues,
				symbols.ValueSlice,
			)
			return nil
		})

		if e == nil {
			methods <- methodName
		}

		pw.CloseWithError(e)
	}

	go write()

	return pr
}

func stringMethods(record marlowRecord, fieldName string, fieldConfig url.Values, methods chan<- string) io.Reader {
	columnName := fieldConfig.Get(constants.ColumnConfigOption)
	methodName := fmt.Sprintf("%sLikeString", columnName)
	likeSuffix := record.config.Get(constants.BlueprintLikeFieldSuffixConfigOption)
	likeFieldName := fmt.Sprintf("%s%s", fieldName, likeSuffix)
	columnReference := fmt.Sprintf("%s.%s", record.table(), columnName)

	symbols := struct {
		PlaceholderSlice string
		ValueItem        string
		ValueSlice       string
		LikeStatement    string
	}{"_placeholders", "_value", "_values", "_like"}

	returns := []string{"string", "[]interface{}"}

	pr, pw := io.Pipe()

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] string LIKE clause for \"%s\"", columnReference)

		e := writer.WithMethod(methodName, record.blueprint(), nil, returns, func(scope url.Values) error {
			likeSlice := fmt.Sprintf("%s.%s", scope.Get("receiver"), likeFieldName)

			writer.WithIf("%s == nil || %s == nil || len(%s) == 0", func(url.Values) error {
				writer.Println("return \"\", nil")
				return nil
			}, scope.Get("receiver"), likeSlice, likeSlice)

			writer.Println("%s := make([]string, 0, len(%s))", symbols.PlaceholderSlice, likeSlice)
			writer.Println("%s := make([]interface{}, 0, len(%s))", symbols.ValueSlice, likeSlice)

			writer.WithIter("_, %s := range %s", func(url.Values) error {
				likeString := fmt.Sprintf("fmt.Sprintf(\"%s LIKE ?\")", columnReference)
				writer.Println("%s := %s", symbols.LikeStatement, likeString)
				writer.Println("%s = append(%s, %s)", symbols.PlaceholderSlice, symbols.PlaceholderSlice, symbols.LikeStatement)
				writer.Println("%s = append(%s, %s)", symbols.ValueSlice, symbols.ValueSlice, symbols.ValueItem)
				return nil
			}, symbols.ValueItem, likeSlice)

			writer.Println("return strings.Join(%s, \" AND \"), %s", symbols.PlaceholderSlice, symbols.ValueSlice)
			return nil
		})

		if e == nil {
			methods <- methodName
		}

		pw.CloseWithError(e)
	}

	go write()

	return pr
}

func numericalMethods(record marlowRecord, fieldName string, fieldConfig url.Values, methods chan<- string) io.Reader {
	columnName := fieldConfig.Get(constants.ColumnConfigOption)
	rangeMethodName := fmt.Sprintf("%sRangeString", columnName)
	rangeFieldName := fmt.Sprintf("%s%s", fieldName, record.config.Get(constants.BlueprintRangeFieldSuffixConfigOption))
	columnReference := fmt.Sprintf("%s.%s", record.table(), columnName)

	pr, pw := io.Pipe()

	returns := []string{"string", "[]interface{}"}

	symbols := struct {
		ValueSlice string
	}{"_values"}

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] range clause methods for %s", columnReference)

		e := writer.WithMethod(rangeMethodName, record.blueprint(), nil, returns, func(scope url.Values) error {
			receiver := scope.Get("receiver")
			rangeArray := fmt.Sprintf("%s.%s", receiver, rangeFieldName)

			writer.WithIf("len(%s) != 2", func(url.Values) error {
				writer.Println("return \"\", nil")
				return nil
			}, rangeArray)

			writer.Println("%s := make([]interface{}, 2)", symbols.ValueSlice)

			writer.Println("%s[0] = %s[0]", symbols.ValueSlice, rangeArray)
			writer.Println("%s[1] = %s[1]", symbols.ValueSlice, rangeArray)

			writer.Println("return \"%s > ? AND %s < ?\", %s", columnReference, columnReference, symbols.ValueSlice)
			return nil
		})

		if e == nil {
			methods <- rangeMethodName
		}

		pw.CloseWithError(e)
	}

	go write()

	return pr
}

// NewBlueprintGenerator returns a reader that will generate the basic query struct type used for record lookups.
func newBlueprintGenerator(record marlowRecord) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		e := writeBlueprint(pw, record)
		pw.CloseWithError(e)
	}()

	return pr
}
