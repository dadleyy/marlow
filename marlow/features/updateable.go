package features

import "io"
import "fmt"
import "net/url"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"

// NewUpdateableGenerator is responsible for generating updating store methods.
func NewUpdateableGenerator(record url.Values, fields map[string]url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		var e error
		gosrc := writing.NewGoWriter(pw)
		updateStructName := fmt.Sprintf("%sUpdates", record.Get("recordName"))

		e = gosrc.WithStruct(updateStructName, func(url.Values) error {
			return nil
		})

		if e != nil {
			pw.CloseWithError(e)
			return
		}

		plural := inflector.Pluralize(record.Get("recordName"))
		methodName := fmt.Sprintf("Update%s", plural)
		receiver := record.Get("storeName")

		bp := blueprint{
			record: record,
			fields: fields,
		}

		params := []writing.FuncParam{
			{Type: fmt.Sprintf("*%s", updateStructName), Symbol: "updates"},
			{Type: fmt.Sprintf("*%s", bp.Name()), Symbol: "blueprint"},
		}

		e = gosrc.WithMethod(methodName, receiver, params, []string{"int", "error"}, func(url.Values) error {
			gosrc.Println("return 0, nil")
			return nil
		})

		pw.CloseWithError(e)
	}()

	return pr
}
