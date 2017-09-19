package features

import "io"
import "fmt"
import "net/url"
import "github.com/dadleyy/marlow/marlow/writing"

func writeStore(destination io.Writer, record url.Values, imports chan<- string) error {
	out := writing.NewGoWriter(destination)

	e := out.WithStruct(record.Get("storeName"), func(url.Values) error {
		out.Println("*sql.DB")
		return nil
	})

	if e != nil {
		return e
	}

	qParams := []writing.FuncParam{
		{Symbol: "_sqlQuery", Type: "string"},
		{Symbol: "_args", Type: "...interface{}"},
	}

	qReturns := []string{"*sql.Rows", "error"}

	e = out.WithMethod("q", record.Get("storeName"), qParams, qReturns, func(scope url.Values) error {
		receiver := scope.Get("receiver")
		condition := fmt.Sprintf("%s.DB == nil || %s.Ping() != nil", receiver, receiver)

		e := out.WithIf(condition, func(url.Values) error {
			out.Println("return nil, fmt.Errorf(\"not-connected\")")
			return nil
		})

		if e != nil {
			return e
		}

		out.Println("return %s.Query(_sqlQuery, _args...)", receiver)
		return nil
	})

	if e != nil {
		return e
	}

	imports <- "fmt"
	imports <- "database/sql"
	return nil
}

func NewStoreGenerator(record url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		e := writeStore(pw, record, imports)
		pw.CloseWithError(e)
	}()

	return pr
}
