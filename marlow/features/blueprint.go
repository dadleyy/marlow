package features

import "io"
import "fmt"
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

			if fieldType == "int" {
				out.Println("%s%s []int", name, bp.record.Get(constants.BlueprintRangeFieldSuffixConfigOption))
			}

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

	var clauseMethods []string

	for name, config := range bp.fields {
		methods, e := fieldMethods(out, bp, name, config)

		if e != nil {
			return e
		}

		clauseMethods = append(clauseMethods, methods...)
	}

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

func fieldMethods(writer writing.GoWriter, p blueprint, name string, config url.Values) ([]string, error) {
	colName, fieldType := config.Get("column"), config.Get("type")
	inString := fmt.Sprintf("%sInString", colName)
	methods := []string{inString}
	tableName := p.record.Get(constants.TableNameConfigOption)

	symbols := map[string]string{
		"VALUE_ARRAY":   "_values",
		"VALUE_ITEM":    "_v",
		"JOINED_VALUES": "_joined",
	}

	writer.WithMethod(inString, p.Name(), nil, []string{"string"}, func(scope url.Values) error {
		fieldReference := fmt.Sprintf("%s.%s", scope.Get("receiver"), name)

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
		writer.Println("return fmt.Sprintf(\"%s.%s IN (%%s)\", %s)", tableName, colName, symbols["JOINED_VALUES"])
		return nil
	})

	var typeMethods []string
	var typeError error

	switch fieldType {
	case "int":
		typeMethods, typeError = numericalMethods(writer, p, name, config)
		break
	case "string":
		typeMethods, typeError = stringMethods(writer, p, name, config)
		break
	}

	if typeError != nil {
		return nil, typeError
	}

	if typeMethods != nil && len(typeMethods) >= 1 {
		methods = append(methods, typeMethods...)
	}

	return methods, nil
}

func stringMethods(writer writing.GoWriter, p blueprint, name string, config url.Values) ([]string, error) {
	colName := config.Get("column")
	likeMethodName := fmt.Sprintf("%sLikeString", colName)
	likeFieldName := fmt.Sprintf("%s%s", name, p.record.Get(constants.BlueprintLikeFieldSuffixConfigOption))
	clauseTarget := fmt.Sprintf("%s.%s", p.record.Get(constants.TableNameConfigOption), colName)

	symbols := map[string]string{
		"VALUE_ITEM":     "_v",
		"VALUE_ARRAY":    "_values",
		"LIKE_STATEMENT": "_statement",
	}

	writer.WithMethod(likeMethodName, p.Name(), nil, []string{"string"}, func(scope url.Values) error {
		likeSlice := fmt.Sprintf("%s.%s", scope.Get("receiver"), likeFieldName)

		writer.WithIf("%s == nil || len(%s) == 0", func(url.Values) error {
			writer.Println("return \"\"")
			return nil
		}, likeSlice, likeSlice)

		writer.Println("%s := make([]string, 0, len(%s))", symbols["VALUE_ARRAY"], likeSlice)

		writer.WithIter("_, %s := range %s", func(url.Values) error {
			likeString := fmt.Sprintf("fmt.Sprintf(\"%s LIKE '%%s'\", %s)", clauseTarget, symbols["VALUE_ITEM"])
			writer.Println("%s := %s", symbols["LIKE_STATEMENT"], likeString)
			writer.Println("%s = append(%s, %s)", symbols["VALUE_ARRAY"], symbols["VALUE_ARRAY"], symbols["LIKE_STATEMENT"])
			return nil
		}, symbols["VALUE_ITEM"], likeSlice)

		writer.Println("return strings.Join(%s, \" AND \")", symbols["VALUE_ARRAY"])
		return nil
	})

	return []string{likeMethodName}, nil
}

func numericalMethods(writer writing.GoWriter, p blueprint, name string, config url.Values) ([]string, error) {
	tableName := p.record.Get(constants.TableNameConfigOption)
	colName := config.Get("column")
	rangeMethodName := fmt.Sprintf("%sRangeString", colName)

	methods := []string{rangeMethodName}
	rangeFieldName := fmt.Sprintf("%s%s", name, p.record.Get(constants.BlueprintRangeFieldSuffixConfigOption))

	writer.WithMethod(rangeMethodName, p.Name(), nil, []string{"string"}, func(scope url.Values) error {
		receiver := scope.Get("receiver")
		rangeArray := fmt.Sprintf("%s.%s", receiver, rangeFieldName)
		clauseTarget := fmt.Sprintf("%s.%s", tableName, colName)

		writer.WithIf("len(%s) != 2", func(url.Values) error {
			writer.Println("return \"\"")
			return nil
		}, rangeArray)

		writer.Println(
			"return fmt.Sprintf(\"%s > %%d AND %s < %%d\", %s[0], %s[1])",
			clauseTarget,
			clauseTarget,
			rangeArray,
			rangeArray,
		)
		return nil
	})

	return methods, nil
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
