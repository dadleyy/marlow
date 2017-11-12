package marlow

import "io"
import "fmt"
import "strings"
import "net/url"
import "github.com/dadleyy/marlow/marlow/writing"

func writeStore(destination io.Writer, record marlowRecord, storeMethods map[string]writing.FuncDecl) error {
	out := writing.NewGoWriter(destination)

	e := out.WithStruct(record.store(), func(url.Values) error {
		out.Println("*sql.DB")
		return nil
	})

	if e != nil {
		return e
	}

	symbols := struct {
		dbParam string
	}{"_db"}

	params := []writing.FuncParam{
		{Type: "*sql.DB", Symbol: symbols.dbParam},
	}

	returns := []string{record.external()}

	e = out.WithFunc(fmt.Sprintf("New%s", record.external()), params, returns, func(url.Values) error {
		out.Println("return &%s{%s}", record.store(), symbols.dbParam)
		return nil
	})

	if e != nil {
		return e
	}

	e = out.WithInterface(record.external(), func(url.Values) error {
		for _, method := range storeMethods {
			params := make([]string, 0)
			returns := strings.Join(method.Returns, ",")

			for _, p := range method.Params {
				params = append(params, fmt.Sprintf("%s", p.Type))
			}

			if len(method.Returns) > 0 {
				returns = fmt.Sprintf("(%s)", returns)
			}

			definition := fmt.Sprintf("%s(%s) %s", method.Name, strings.Join(params, ","), returns)
			out.Println(definition)
		}
		return nil
	})

	record.registerImports("database/sql")
	return e
}

// newStoreGenerator returns a reader that will generate the centralized record store for a given record.
func newStoreGenerator(record marlowRecord, methods map[string]writing.FuncDecl) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		e := writeStore(pw, record, methods)
		pw.CloseWithError(e)
	}()

	return pr
}
