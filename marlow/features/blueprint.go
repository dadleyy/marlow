package features

import "io"
import "fmt"
import "net/url"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"

type blueprint struct {
	record url.Values
	fields map[string]url.Values
}

func (p blueprint) Name() string {
	singular := inflector.Singularize(p.record.Get("recordName"))
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

	imports <- "strings"

	symbols := map[string]string{
		"CLAUSE_ARRAY": "_clauseArray",
	}

	return out.WithMethod("String", bp.Name(), nil, []string{"string"}, func(scope url.Values) error {
		out.Println("%s := make([]string, 0)", symbols["CLAUSE_ARRAY"])

		if e := writeBlueprintFieldConditionals(out, bp, scope.Get("receiver"), symbols["CLAUSE_ARRAY"]); e != nil {
			return e
		}

		out.WithIf("len(%s) == 0", func(url.Values) error {
			out.Println("return \"\"")
			return nil
		}, symbols["CLAUSE_ARRAY"])

		out.Println("return strings.Join(%s, \" AND \")", symbols["CLAUSE_ARRAY"])
		return nil
	})
}

func writeBlueprintFieldConditionals(w writing.GoWriter, p blueprint, receiver string, list string) error {
	symbols := map[string]string{
		"VALUE_ARRAY": "_values",
		"VALUE_ITEM":  "_v",
	}

	for name, config := range p.fields {
		colName, tableName := config.Get("column"), p.record.Get("tableName")
		fieldReference := fmt.Sprintf("%s.%s", receiver, name)

		w.WithIf("len(%s) >= 1", func(url.Values) error {
			w.Println("%s := make([]string, 0, len(%s))", symbols["VALUE_ARRAY"], fieldReference)

			w.WithIter("_, %s := range %s", func(url.Values) error {
				w.Println(
					"%s = append(%s, fmt.Sprintf(\"'%%v'\", %s))",
					symbols["VALUE_ARRAY"],
					symbols["VALUE_ARRAY"],
					symbols["VALUE_ITEM"],
				)

				return nil
			}, symbols["VALUE_ITEM"], fieldReference)

			w.Println(
				"%s = append(%s, fmt.Sprintf(\"WHERE %s.%s IN (%%s)\", strings.Join(%s, \",\")))",
				list,
				list,
				tableName,
				colName,
				symbols["VALUE_ARRAY"],
			)

			return nil
		}, fieldReference)
	}

	return nil
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
