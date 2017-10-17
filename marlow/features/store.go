package features

import "io"
import "fmt"
import "net/url"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

func writeStore(destination io.Writer, record url.Values, imports chan<- string) error {
	out := writing.NewGoWriter(destination)
	storeName := record.Get(constants.StoreNameConfigOption)

	e := out.WithStruct(storeName, func(url.Values) error {
		out.Println("*sql.DB")
		return nil
	})

	if e != nil {
		return e
	}

	symbols := map[string]string{
		"STATEMENT_PARAM": "_statement",
		"ARGUMENTS_PARAM": "_args",
	}

	qParams := []writing.FuncParam{
		{Symbol: symbols["STATEMENT_PARAM"], Type: "string"},
		{Symbol: symbols["ARGUMENTS_PARAM"], Type: "...interface{}"},
	}

	qReturns := []string{"*sql.Rows", "error"}
	eReturns := []string{"sql.Result", "error"}

	e = out.WithMethod("q", storeName, qParams, qReturns, func(scope url.Values) error {
		receiver := scope.Get("receiver")
		condition := fmt.Sprintf("%s.DB == nil || %s.Ping() != nil", receiver, receiver)

		e := out.WithIf(condition, func(url.Values) error {
			out.Println("return nil, fmt.Errorf(\"not-connected\")")
			return nil
		})

		if e != nil {
			return e
		}

		out.Println("return %s.Query(%s, %s...)", receiver, symbols["STATEMENT_PARAM"], symbols["ARGUMENTS_PARAM"])
		return nil
	})

	if e != nil {
		return e
	}

	e = out.WithMethod("e", storeName, qParams, eReturns, func(scope url.Values) error {
		receiver := scope.Get("receiver")
		condition := fmt.Sprintf("%s.DB == nil || %s.Ping() != nil", receiver, receiver)

		e := out.WithIf(condition, func(url.Values) error {
			out.Println("return nil, fmt.Errorf(\"not-connected\")")
			return nil
		})

		if e != nil {
			return e
		}

		out.Println("return %s.Exec(%s, %s...)", receiver, symbols["STATEMENT_PARAM"], symbols["ARGUMENTS_PARAM"])
		return nil
	})

	if e != nil {
		return e
	}

	imports <- "fmt"
	imports <- "database/sql"
	return nil
}

// NewStoreGenerator returns a reader that will generate the centralized record store for a given record.
func NewStoreGenerator(record url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		e := writeStore(pw, record, imports)
		pw.CloseWithError(e)
	}()

	return pr
}
