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

	return out.WithMethod("String", bp.Name(), nil, []string{"string"}, func(url.Values) error {
		out.Println("return strings.ToLower(\"\")")
		return nil
	})
}

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
