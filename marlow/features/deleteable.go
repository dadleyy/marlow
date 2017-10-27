package features

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

// NewDeleteableGenerator is responsible for creating a generator that will write out the Delete api methods.
func NewDeleteableGenerator(record url.Values, fields map[string]url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()

	storeName := record.Get(constants.StoreNameConfigOption)
	recordName := record.Get(constants.RecordNameConfigOption)
	methodName := fmt.Sprintf("Delete%s", inflector.Pluralize(recordName))
	tableName := record.Get(constants.TableNameConfigOption)

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
		{Type: fmt.Sprintf("*%s", blueprint{record: record}.Name()), Symbol: symbols.BlueprintParam},
	}

	returns := []string{
		"int64",
		"error",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)

		gosrc.Comment("[marlow] deleteable")

		e := gosrc.WithMethod(methodName, storeName, params, returns, func(scope url.Values) error {
			gosrc.WithIf("%s == nil || %s.String() == \"\"", func(url.Values) error {
				gosrc.Println("return -1, fmt.Errorf(\"%s\")", constants.InvalidDeletionBlueprint)
				return nil
			}, symbols.BlueprintParam, symbols.BlueprintParam)

			deleteString := fmt.Sprintf("DELETE FROM %s", tableName)

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
			imports <- "fmt"
		}

		pw.CloseWithError(e)
	}()

	return pr
}
