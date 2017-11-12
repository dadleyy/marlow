package marlow

import "io"
import "fmt"
import "net/url"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type deleteableSymbols struct {
	Error           string
	RowCount        string
	BlueprintParam  string
	ExecResult      string
	ExecError       string
	StatementString string
	StatementResult string
	StatementError  string
}

// newDeleteableGenerator is responsible for creating a generator that will write out the Delete api methods.
func newDeleteableGenerator(record marlowRecord) io.Reader {
	pr, pw := io.Pipe()
	methodName := fmt.Sprintf("Delete%s", inflector.Pluralize(record.name()))

	symbols := deleteableSymbols{
		Error:           "_e",
		RowCount:        "_count",
		BlueprintParam:  "_blueprint",
		StatementString: "_query",
		StatementResult: "_statement",
		StatementError:  "_se",
		ExecResult:      "_execResult",
		ExecError:       "_ee",
	}

	params := []writing.FuncParam{
		{Type: fmt.Sprintf("*%s", record.blueprint()), Symbol: symbols.BlueprintParam},
	}

	returns := []string{
		"int64",
		"error",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)

		gosrc.Comment("[marlow] deleteable")

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			gosrc.WithIf("%s == nil || %s.String() == \"\"", func(url.Values) error {
				gosrc.Println("return -1, fmt.Errorf(\"%s\")", constants.InvalidDeletionBlueprint)
				return nil
			}, symbols.BlueprintParam, symbols.BlueprintParam)

			deleteString := fmt.Sprintf("DELETE FROM %s", record.table())

			gosrc.Println(
				"%s := fmt.Sprintf(\"%s %%s\", %s)",
				symbols.StatementString,
				deleteString,
				symbols.BlueprintParam,
			)

			gosrc.Println(
				"%s, %s := %s.Prepare(%s + \";\")",
				symbols.StatementResult,
				symbols.Error,
				scope.Get("receiver"),
				symbols.StatementString,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.Error)
				return nil
			}, symbols.Error)

			gosrc.Println("defer %s.Close()", symbols.StatementResult)

			gosrc.Println(
				"%s, %s := %s.Exec(%s.Values()...)",
				symbols.ExecResult,
				symbols.ExecError,
				symbols.StatementResult,
				symbols.BlueprintParam,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.ExecError)
				return nil
			}, symbols.ExecError)

			gosrc.Println("%s, %s := %s.RowsAffected()", symbols.RowCount, symbols.Error, symbols.ExecResult)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols.Error)
				return nil
			}, symbols.Error)

			gosrc.Println("return %s, nil", symbols.RowCount)

			return nil
		})

		if e == nil {
			record.registerImports("fmt")
		}

		pw.CloseWithError(e)
	}()

	return pr
}
