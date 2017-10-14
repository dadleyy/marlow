package features

import "io"
import "fmt"
import "sync"
import "strings"
import "net/url"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type blueprint struct {
	record url.Values
	fields map[string]url.Values
}

func (p blueprint) Name() string {
	singular := inflector.Singularize(p.record.Get(constants.RecordNameConfigOption))
	return fmt.Sprintf("%sBlueprint", singular)
}

func writeBlueprint(destination io.Writer, bp blueprint, imports chan<- string) error {
	out := writing.NewGoWriter(destination)

	e := out.WithStruct(bp.Name(), func(url.Values) error {
		for name, config := range bp.fields {
			fieldType := config.Get("type")

			if fieldType == "" {
				return fmt.Errorf("bad field type for field name: %s", name)
			}

			// Support IN lookup on string fields.
			if fieldType == "int" {
				out.Println("%s%s []int", name, bp.record.Get(constants.BlueprintRangeFieldSuffixConfigOption))
			}

			// Support LIKE lookup on string fields.
			if fieldType == "string" {
				out.Println("%s%s []string", name, bp.record.Get(constants.BlueprintLikeFieldSuffixConfigOption))
			}

			if fieldImport := config.Get("import"); fieldImport != "" {
				imports <- fieldImport
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

	imports <- "fmt"
	imports <- "strings"

	symbols := map[string]string{
		"CLAUSE_ARRAY": "_clauseArray",
		"CLAUSE_ITEM":  "_clauseItem",
	}

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

	for name, config := range bp.fields {
		fieldGenerators := fieldMethods(bp, name, config, methodReceiver)

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

	// With all of our fields having generated non-exported clause generation methods on our struct, we can create the
	// final 'String' method which iterates over all of these, calling them and adding the non-empty string clauses to
	// a list, which eventually is returned as a joined string.
	return out.WithMethod("String", bp.Name(), nil, []string{"string"}, func(scope url.Values) error {
		out.Println("%s := make([]string, 0)", symbols["CLAUSE_ARRAY"])

		for _, method := range clauseMethods {
			out.WithIf("%s := %s.%s(); %s != \"\"", func(url.Values) error {
				out.Println("%s = append(%s, %s)", symbols["CLAUSE_ARRAY"], symbols["CLAUSE_ARRAY"], symbols["CLAUSE_ITEM"])
				return nil
			}, symbols["CLAUSE_ITEM"], scope.Get("receiver"), method, symbols["CLAUSE_ITEM"])
		}

		out.WithIf("len(%s) == 0", func(url.Values) error {
			out.Println("return \"\"")
			return nil
		}, symbols["CLAUSE_ARRAY"])

		out.Println("return \"WHERE \" + strings.Join(%s, \" AND \")", symbols["CLAUSE_ARRAY"])
		return nil
	})
}

func fieldMethods(print blueprint, name string, config url.Values, methods chan<- string) []io.Reader {
	fieldType := config.Get("type")
	results := make([]io.Reader, 0, len(print.fields))

	if fieldType == "string" || fieldType == "int" {
		results = append(results, simpleTypeIn(print, name, config, methods))
	}

	if fieldType == "string" {
		results = append(results, stringMethods(print, name, config, methods))
	}

	if fieldType == "int" {
		results = append(results, numericalMethods(print, name, config, methods))
	}

	if fieldType == "sql.NullInt64" {
		results = append(results, nullableIntMethods(print, name, config, methods))
	}

	if len(results) == 0 {
		warning := fmt.Sprintf("// [marlow] field %s (%s) unable to generate clauses. unsupported type", name, fieldType)

		results = []io.Reader{strings.NewReader(warning)}
	}

	return results
}

func nullableIntMethods(print blueprint, fieldName string, config url.Values, methods chan<- string) io.Reader {
	pr, pw := io.Pipe()
	columnName := config.Get(constants.ColumnConfigOption)
	tableName := print.record.Get(constants.TableNameConfigOption)
	methodName := fmt.Sprintf("%sInString", columnName)

	symbols := map[string]string{
		"VALUE_ARRAY":   "_values",
		"VALUE_ITEM":    "_v",
		"JOINED_VALUES": "_joined",
	}

	columnReference := fmt.Sprintf("%s.%s", tableName, columnName)

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] nullable clause gen for \"%s\"", columnReference)

		e := writer.WithMethod(methodName, print.Name(), nil, []string{"string"}, func(scope url.Values) error {
			fieldReference := fmt.Sprintf("%s.%s", scope.Get("receiver"), fieldName)

			// Add conditional check for length presence on lookup slice.
			writer.WithIf("%s == nil", func(url.Values) error {
				writer.Println("return \"\"")
				return nil
			}, fieldReference)

			// Add conditional check for length presence on lookup slice.
			writer.WithIf("len(%s) == 0", func(url.Values) error {
				writer.Println("return \"%s NOT NULL\"", columnReference)
				return nil
			}, fieldReference)

			writer.Println("%s := make([]string, 0, len(%s))", symbols["VALUE_ARRAY"], fieldReference)

			writer.WithIter("_, %s := range %s", func(url.Values) error {

				writer.WithIf("%s.Valid == false", func(url.Values) error {
					writer.Println("return \"%s IS NULL\"", columnReference)
					return nil
				}, symbols["VALUE_ITEM"])

				writer.Println(
					"%s = append(%s, fmt.Sprintf(\"'%%v'\", %s.Int64))",
					symbols["VALUE_ARRAY"],
					symbols["VALUE_ARRAY"],
					symbols["VALUE_ITEM"],
				)

				return nil
			}, symbols["VALUE_ITEM"], fieldReference)

			writer.Println("%s := strings.Join(%s, \",\")", symbols["JOINED_VALUES"], symbols["VALUE_ARRAY"])
			writer.Println("return fmt.Sprintf(\"%s.%s IN (%%s)\", %s)", tableName, columnName, symbols["JOINED_VALUES"])

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

func simpleTypeIn(print blueprint, fieldName string, config url.Values, methods chan<- string) io.Reader {
	pr, pw := io.Pipe()
	columnName := config.Get(constants.ColumnConfigOption)
	tableName := print.record.Get(constants.TableNameConfigOption)
	methodName := fmt.Sprintf("%sInString", columnName)
	columnReference := fmt.Sprintf("%s.%s", tableName, columnName)

	symbols := map[string]string{
		"VALUE_ARRAY":   "_values",
		"VALUE_ITEM":    "_v",
		"JOINED_VALUES": "_joined",
	}

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] type IN clause for \"%s\"", columnReference)

		e := writer.WithMethod(methodName, print.Name(), nil, []string{"string"}, func(scope url.Values) error {
			fieldReference := fmt.Sprintf("%s.%s", scope.Get("receiver"), fieldName)

			// Add conditional check for length presence on lookup slice.
			writer.WithIf("len(%s) == 0", func(url.Values) error {
				writer.Println("return \"\"")
				return nil
			}, fieldReference)

			writer.Println("%s := make([]string, 0, len(%s))", symbols["VALUE_ARRAY"], fieldReference)

			writer.WithIter("_, %s := range %s", func(url.Values) error {
				writer.Println(
					"%s = append(%s, fmt.Sprintf(\"'%%v'\", %s))",
					symbols["VALUE_ARRAY"],
					symbols["VALUE_ARRAY"],
					symbols["VALUE_ITEM"],
				)
				return nil
			}, symbols["VALUE_ITEM"], fieldReference)

			writer.Println("%s := strings.Join(%s, \",\")", symbols["JOINED_VALUES"], symbols["VALUE_ARRAY"])
			writer.Println("return fmt.Sprintf(\"%s IN (%%s)\", %s)", columnReference, symbols["JOINED_VALUES"])
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

func stringMethods(print blueprint, name string, config url.Values, methods chan<- string) io.Reader {
	columnName := config.Get(constants.ColumnConfigOption)
	methodName := fmt.Sprintf("%sLikeString", columnName)
	likeFieldName := fmt.Sprintf("%s%s", name, print.record.Get(constants.BlueprintLikeFieldSuffixConfigOption))
	columnReference := fmt.Sprintf("%s.%s", print.record.Get(constants.TableNameConfigOption), columnName)

	symbols := map[string]string{
		"VALUE_ITEM":     "_v",
		"VALUE_ARRAY":    "_values",
		"LIKE_STATEMENT": "_statement",
	}

	pr, pw := io.Pipe()

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] string LIKE clause for \"%s\"", columnReference)

		e := writer.WithMethod(methodName, print.Name(), nil, []string{"string"}, func(scope url.Values) error {
			likeSlice := fmt.Sprintf("%s.%s", scope.Get("receiver"), likeFieldName)

			writer.WithIf("%s == nil || len(%s) == 0", func(url.Values) error {
				writer.Println("return \"\"")
				return nil
			}, likeSlice, likeSlice)

			writer.Println("%s := make([]string, 0, len(%s))", symbols["VALUE_ARRAY"], likeSlice)

			writer.WithIter("_, %s := range %s", func(url.Values) error {
				likeString := fmt.Sprintf("fmt.Sprintf(\"%s LIKE '%%s'\", %s)", columnReference, symbols["VALUE_ITEM"])
				writer.Println("%s := %s", symbols["LIKE_STATEMENT"], likeString)
				writer.Println("%s = append(%s, %s)", symbols["VALUE_ARRAY"], symbols["VALUE_ARRAY"], symbols["LIKE_STATEMENT"])
				return nil
			}, symbols["VALUE_ITEM"], likeSlice)

			writer.Println("return strings.Join(%s, \" AND \")", symbols["VALUE_ARRAY"])
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

func numericalMethods(print blueprint, name string, config url.Values, methods chan<- string) io.Reader {
	tableName := print.record.Get(constants.TableNameConfigOption)
	columnName := config.Get(constants.ColumnConfigOption)
	rangeMethodName := fmt.Sprintf("%sRangeString", columnName)
	rangeFieldName := fmt.Sprintf("%s%s", name, print.record.Get(constants.BlueprintRangeFieldSuffixConfigOption))
	columnReference := fmt.Sprintf("%s.%s", tableName, columnName)

	pr, pw := io.Pipe()

	write := func() {
		writer := writing.NewGoWriter(pw)
		writer.Comment("[marlow] range clause methods for %s", columnReference)

		e := writer.WithMethod(rangeMethodName, print.Name(), nil, []string{"string"}, func(scope url.Values) error {
			receiver := scope.Get("receiver")
			rangeArray := fmt.Sprintf("%s.%s", receiver, rangeFieldName)

			writer.WithIf("len(%s) != 2", func(url.Values) error {
				writer.Println("return \"\"")
				return nil
			}, rangeArray)

			writer.Println(
				"return fmt.Sprintf(\"%s > %%d AND %s < %%d\", %s[0], %s[1])",
				columnReference,
				columnReference,
				rangeArray,
				rangeArray,
			)
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
func NewBlueprintGenerator(record url.Values, fields map[string]url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		bp := blueprint{
			record: record,
			fields: fields,
		}

		e := writeBlueprint(pw, bp, imports)
		pw.CloseWithError(e)
	}()

	return pr
}
