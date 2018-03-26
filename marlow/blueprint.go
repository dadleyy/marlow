package marlow

import "io"
import "fmt"
import "sync"
import "strings"
import "net/url"
import "go/types"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

func writeBlueprintStruct(out writing.GoWriter, record marlowRecord) error {
	e := out.WithStruct(record.blueprint(), func(url.Values) error {
		for name, config := range record.fields {
			fieldType := config.Get("type")

			if fieldType == "" {
				return fmt.Errorf("bad field type for field name: %s", name)
			}

			typeInfo := getTypeInfo(fieldType)

			// Support IN lookup on string fields.
			if typeInfo&types.IsNumeric != 0 {
				out.Println("%s%s []%s", name, record.config.Get(constants.BlueprintRangeFieldSuffixConfigOption), fieldType)
			}

			// Support LIKE lookup on string fields.
			if typeInfo&types.IsString != 0 {
				out.Println("%s%s []string", name, record.config.Get(constants.BlueprintLikeFieldSuffixConfigOption))
			}

			if fieldImport := config.Get("import"); fieldImport != "" {
				record.registerImports(fieldImport)
			}

			out.Println("%s []%s", name, fieldType)
		}

		if deletion := record.deletionField(); deletion != nil {
			out.Println("Unscoped bool")
		}

		out.Println("Inclusive bool")
		out.Println("Limit int")
		out.Println("Offset int")
		out.Println("OrderBy string")
		out.Println("OrderDirection string")

		return nil
	})

	return e
}

func writeBlueprint(destination io.Writer, record marlowRecord) error {
	out := writing.NewGoWriter(destination)

	if e := writeBlueprintStruct(out, record); e != nil {
		return e
	}

	record.registerImports("fmt", "strings")

	var readers []io.Reader
	methodReceiver := make(chan string)
	var clauseMethods []string
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Capture all generated string methods that are used to create a clause for specific fields.
	go func() {
		for name := range methodReceiver {
			clauseMethods = append(clauseMethods, name)
		}
		wg.Done()
	}()

	// Loop over every field, creating the generators that will be used to create their string-producing clause methods.
	for _, f := range record.fieldList(nil) {
		name, config := f.name, record.fields[f.name]
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
		clauseMap   string
		clauseSlice string
		clauseItem  string
		valueCount  string
		values      string
	}{"_map", "_clauses", "_item", "_count", "_values"}

	// With all of our fields having generated non-exported clause generation methods on our struct, we can create the
	// final 'String' method which iterates over all of these, calling them and adding the non-empty string clauses to
	// a list, which eventually is returned as a joined string.
	e := out.WithMethod("String", record.blueprint(), nil, []string{"string"}, func(scope url.Values) error {
		out.Println("%s := make([]string, 0, %d)", symbols.clauseSlice, len(clauseMethods))
		out.Println("%s := 1", symbols.valueCount)

		for _, method := range clauseMethods {
			out.WithIf("%s, %s := %s.%s(%s); %s != \"\"", func(url.Values) error {
				out.Println("%s = append(%s, %s)", symbols.clauseSlice, symbols.clauseSlice, symbols.clauseItem)
				out.Println("%s+=len(%s)", symbols.valueCount, symbols.values)
				return nil
			}, symbols.clauseItem, symbols.values, scope.Get("receiver"), method, symbols.valueCount, symbols.clauseItem)
		}

		out.WithIf("len(%s) == 0", func(url.Values) error {
			return out.Returns(writing.EmptyString)
		}, symbols.clauseSlice)

		out.Println("%s := \" AND \"", symbols.clauseMap)

		out.WithIf("%s.Inclusive == true", func(url.Values) error {
			return out.Println("%s = \" OR \"", symbols.clauseMap)
		}, scope.Get("receiver"))

		query := fmt.Sprintf("\"WHERE \" + strings.Join(%s, %s)", symbols.clauseSlice, symbols.clauseMap)

		deletion := record.deletionField()
		if deletion == nil {
			return out.Returns(query)
		}

		selector := fmt.Sprintf("%s.%s", record.table(), deletion.Get(constants.ColumnConfigOption))

		out.WithIf("%s != nil && %s.Unscoped == true", func(url.Values) error {
			return out.Returns(query)
		}, scope.Get("receiver"), scope.Get("receiver"))

		query += fmt.Sprintf("+ \" AND %s IS NULL\"", selector)
		return out.Returns(query)
	})

	if e != nil {
		return e
	}

	return out.WithMethod("Values", record.blueprint(), nil, []string{"[]interface{}"}, func(scope url.Values) error {
		out.Println("%s := make([]interface{}, 0, %d)", symbols.clauseSlice, len(clauseMethods))

		out.WithIf("%s == nil", func(url.Values) error {
			return out.Returns(writing.Nil)
		}, scope.Get("receiver"))

		for _, method := range clauseMethods {
			out.WithIf("_, %s := %s.%s(0); %s != nil && len(%s) > 0", func(url.Values) error {
				return out.Println("%s = append(%s, %s...)", symbols.clauseSlice, symbols.clauseSlice, symbols.clauseItem)
			}, symbols.clauseItem, scope.Get("receiver"), method, symbols.clauseItem, symbols.clauseItem)
		}

		return out.Returns(symbols.clauseSlice)
	})
}

func fieldMethods(record marlowRecord, name string, config url.Values, methods chan<- string) []io.Reader {
	fieldType := config.Get("type")
	results := make([]io.Reader, 0, len(record.fields))
	typeInfo := getTypeInfo(fieldType)

	if typeInfo&types.IsConstType != 0 {
		results = append(results, simpleTypeIn(record, name, config, methods))
	}

	if typeInfo&types.IsString != 0 {
		results = append(results, stringMethods(record, name, config, methods))
	}

	if typeInfo&types.IsNumeric != 0 {
		results = append(results, numericalMethods(record, name, config, methods))
	}

	if fieldType == "sql.NullInt64" {
		results = append(results, nullableIntMethods(record, name, config, methods))
	}

	if len(results) == 0 {
		warning := fmt.Sprintf("/* [marlow] %s (%s) unsupported type %b */\n\n", name, fieldType, typeInfo)
		results = []io.Reader{strings.NewReader(warning)}
	}

	return results
}

func nullableIntMethods(record marlowRecord, fieldName string, config url.Values, methods chan<- string) io.Reader {
	pr, pw := io.Pipe()
	columnName := config.Get(constants.ColumnConfigOption)
	methodName := fmt.Sprintf("%sInString", columnName)

	symbols := struct {
		placeholders string
		values       string
		item         string
		result       string
		valueCount   string
		index        string
	}{"_placeholders", "_values", "_v", "_joined", "_count", "_"}

	columnReference := fmt.Sprintf("%s.%s", record.table(), columnName)

	returns := []string{"string", "[]interface{}"}
	params := []writing.FuncParam{
		{Type: "int", Symbol: symbols.valueCount},
	}

	if record.dialect() == "postgres" {
		symbols.index = "_i"
	}

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] nullable clause gen for \"%s\"", columnReference)

		e := writer.WithMethod(methodName, record.blueprint(), params, returns, func(scope url.Values) error {
			fieldReference := fmt.Sprintf("%s.%s", scope.Get("receiver"), fieldName)

			// Add conditional check for length presence on lookup slice.
			writer.WithIf("%s == nil", func(url.Values) error {
				return writer.Returns(writing.EmptyString, writing.Nil)
			}, fieldReference)

			// Add conditional check for length presence on lookup slice.
			writer.WithIf("len(%s) == 0", func(url.Values) error {
				query := fmt.Sprintf("\"%s NOT NULL\"", columnReference)

				if record.dialect() == "postgres" {
					query = fmt.Sprintf("\"%s IS NOT NULL\"", columnReference)
				}

				return writer.Returns(query, writing.Nil)
			}, fieldReference)

			writer.Println("%s := make([]string, 0, len(%s))", symbols.placeholders, fieldReference)
			writer.Println("%s := make([]interface{}, 0, len(%s))", symbols.values, fieldReference)

			writer.WithIter("%s, %s := range %s", func(url.Values) error {
				writer.WithIf("%s.Valid == false", func(url.Values) error {
					return writer.Returns(fmt.Sprintf("\"%s IS NULL\"", columnReference), writing.Nil)
				}, symbols.item)

				// TODO: cleanup dialog placeholder generation...
				switch record.dialect() {
				case "postgres":
					placeholderString := "%s = append(%s, fmt.Sprintf(\"$%%d\", %s+%s))"
					writer.Println(
						placeholderString,
						symbols.placeholders,
						symbols.placeholders,
						symbols.index,
						symbols.valueCount,
					)
				default:
					writer.Println("%s = append(%s, \"?\")", symbols.placeholders, symbols.placeholders)
				}
				writer.Println("%s = append(%s, %s)", symbols.values, symbols.values, symbols.item)
				return nil
			}, symbols.index, symbols.item, fieldReference)

			writer.Println("%s := strings.Join(%s, \",\")", symbols.result, symbols.placeholders)

			clauseString := fmt.Sprintf("fmt.Sprintf(\"%s.%s IN (%%s)\", %s)", record.table(), columnName, symbols.result)
			return writer.Returns(clauseString, symbols.values)
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
		placeholders    string
		values          string
		item            string
		result          string
		counter         string
		index           string
		placeholderItem string
	}{"_placeholder", "_values", "_v", "_joined", "_count", "_i", "_p"}

	returns := []string{"string", "[]interface{}"}
	params := []writing.FuncParam{
		{Type: "int", Symbol: symbols.counter},
	}

	if record.dialect() != "postgres" {
		symbols.index = "_"
	}

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] type IN clause for \"%s\"", columnReference)

		e := writer.WithMethod(methodName, record.blueprint(), params, returns, func(scope url.Values) error {
			fieldReference := fmt.Sprintf("%s.%s", scope.Get("receiver"), fieldName)

			// Add conditional check for length presence on lookup slice.
			writer.WithIf("len(%s) == 0", func(url.Values) error {
				return writer.Returns(writing.EmptyString, writing.Nil)
			}, fieldReference)

			writer.Println("%s := make([]string, 0, len(%s))", symbols.placeholders, fieldReference)
			writer.Println("%s := make([]interface{}, 0, len(%s))", symbols.values, fieldReference)

			writer.WithIter("%s, %s := range %s", func(url.Values) error {
				// TODO: cleanup dialog placeholder generation...
				switch record.dialect() {
				case "postgres":
					writer.Println("%s := fmt.Sprintf(\"$%%d\", %s+%s)", symbols.placeholderItem, symbols.index, symbols.counter)
					writer.Println("%s = append(%s, %s)", symbols.placeholders, symbols.placeholders, symbols.placeholderItem)
				default:
					writer.Println("%s = append(%s, \"?\")", symbols.placeholders, symbols.placeholders)
				}

				writer.Println("%s = append(%s, %s)", symbols.values, symbols.values, symbols.item)
				return nil
			}, symbols.index, symbols.item, fieldReference)

			writer.Println("%s := strings.Join(%s, \",\")", symbols.result, symbols.placeholders)
			clauseString := fmt.Sprintf("fmt.Sprintf(\"%s IN (%%s)\", %s)", columnReference, symbols.result)
			return writer.Returns(clauseString, symbols.values)
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
		conjunction  string
		placeholders string
		item         string
		values       string
		statement    string
		count        string
		index        string
	}{"_conjunc", "_placeholders", "_value", "_values", "_like", "_count", "_i"}

	if record.dialect() != "postgres" {
		symbols.index = "_"
	}

	returns := []string{"string", "[]interface{}"}
	params := []writing.FuncParam{
		{Type: "int", Symbol: symbols.count},
	}

	pr, pw := io.Pipe()

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] string LIKE clause for \"%s\"", columnReference)

		e := writer.WithMethod(methodName, record.blueprint(), params, returns, func(scope url.Values) error {
			likeSlice := fmt.Sprintf("%s.%s", scope.Get("receiver"), likeFieldName)

			writer.WithIf("%s == nil || %s == nil || len(%s) == 0", func(url.Values) error {
				return writer.Returns(writing.EmptyString, writing.Nil)
			}, scope.Get("receiver"), likeSlice, likeSlice)

			writer.Println("%s := make([]string, 0, len(%s))", symbols.placeholders, likeSlice)
			writer.Println("%s := make([]interface{}, 0, len(%s))", symbols.values, likeSlice)

			writer.WithIter("%s, %s := range %s", func(url.Values) error {
				likeString := fmt.Sprintf("\"%s LIKE ?\"", columnReference)

				if record.dialect() == "postgres" {
					psqlLike := "fmt.Sprintf(\"%s LIKE $%%d\", %s+%s)"
					likeString = fmt.Sprintf(psqlLike, columnReference, symbols.count, symbols.index)
				}

				writer.Println("%s := %s", symbols.statement, likeString)
				writer.Println("%s = append(%s, %s)", symbols.placeholders, symbols.placeholders, symbols.statement)
				return writer.Println("%s = append(%s, %s)", symbols.values, symbols.values, symbols.item)
			}, symbols.index, symbols.item, likeSlice)

			writer.Println("%s := \" AND \"", symbols.conjunction)

			writer.WithIf("%s.Inclusive == true", func(url.Values) error {
				return writer.Println("%s = \" OR \"", symbols.conjunction)
			}, scope.Get("receiver"))

			clauseString := fmt.Sprintf("strings.Join(%s, %s)", symbols.placeholders, symbols.conjunction)
			return writer.Returns(clauseString, symbols.values)
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
		values string
		count  string
	}{"_values", "_count"}

	params := []writing.FuncParam{
		{Type: "int", Symbol: symbols.count},
	}

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] range clause methods for %s", columnReference)

		e := writer.WithMethod(rangeMethodName, record.blueprint(), params, returns, func(scope url.Values) error {
			receiver := scope.Get("receiver")
			rangeArray := fmt.Sprintf("%s.%s", receiver, rangeFieldName)

			writer.WithIf("len(%s) != 2", func(url.Values) error {
				return writer.Returns(writing.EmptyString, writing.Nil)
			}, rangeArray)

			writer.Println("%s := make([]interface{}, 2)", symbols.values)

			writer.Println("%s[0] = %s[0]", symbols.values, rangeArray)
			writer.Println("%s[1] = %s[1]", symbols.values, rangeArray)

			if record.dialect() == "postgres" {
				rangeString := fmt.Sprintf(
					"fmt.Sprintf(\"(%s > $%%d AND %s < $%%d)\", %s, %s+1)",
					columnReference,
					columnReference,
					symbols.count,
					symbols.count,
				)

				return writer.Returns(rangeString, symbols.values)
			}

			return writer.Returns(fmt.Sprintf("\"(%s > ? AND %s < ?)\"", columnReference, columnReference), symbols.values)
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
