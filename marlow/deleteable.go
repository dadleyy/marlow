package marlow

import "io"
import "fmt"
import "net/url"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type deleteableSymbols struct {
	e              string
	count          string
	blueprint      string
	result         string
	statement      string
	prepared       string
	statementError string
}

// newDeleteableGenerator is responsible for creating a generator that will write out the Delete api methods.
func newDeleteableGenerator(record marlowRecord) io.Reader {
	pr, pw := io.Pipe()
	methodName := fmt.Sprintf("Delete%s", inflector.Pluralize(record.name()))

	symbols := deleteableSymbols{
		e:              "_e",
		count:          "_count",
		blueprint:      "_blueprint",
		statement:      "_query",
		prepared:       "_statement",
		statementError: "_se",
		result:         "_execResult",
	}

	params := []writing.FuncParam{
		{Type: fmt.Sprintf("*%s", record.blueprint()), Symbol: symbols.blueprint},
	}

	// Check to see if we have a soft delete configured on the record and validate that the field actually exists.
	if target := record.config.Get(constants.RecordSoftDeleteConfigOption); target != "" {
		field, ok := record.fields[target]

		if !ok {
			pw.CloseWithError(fmt.Errorf("specified soft delete field %s but unable to find source field", target))
			return pr
		}

		pw.Close()

		// Since we're really doing an update, create an updater generator.
		return updater(record, field, &updateOperation{
			name:      methodName,
			operation: "NOW()",
			valueless: true,
		})
	}

	returns := []string{
		"int64",
		"error",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)

		gosrc.Comment("[marlow] deleteable")

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			receiver := scope.Get("receiver")
			logwriter := logWriter{receiver: receiver, output: gosrc}

			gosrc.WithIf("%s == nil || %s.String() == \"\"", func(url.Values) error {
				return gosrc.Returns("-1", fmt.Sprintf("fmt.Errorf(\"%s\")", constants.InvalidDeletionBlueprint))
			}, symbols.blueprint, symbols.blueprint)

			deleteString := fmt.Sprintf("DELETE FROM %s", record.table())

			gosrc.Println("%s := fmt.Sprintf(\"%s %%s\", %s)", symbols.statement, deleteString, symbols.blueprint)
			gosrc.Println("%s, %s := %s.Prepare(%s + \";\")", symbols.prepared, symbols.e, receiver, symbols.statement)

			// Check for preparation error.
			gosrc.WithIf("%s != nil", func(url.Values) error { return gosrc.Returns("-1", symbols.e) }, symbols.e)

			// Always close the prepared statement.
			gosrc.Println("defer %s.Close()", symbols.prepared)

			logwriter.AddLog(symbols.statement, fmt.Sprintf("%s.Values()", symbols.blueprint))

			// Executre the prepared statement with the values from the blueprint.
			gosrc.Println(
				"%s, %s := %s.Exec(%s.Values()...)",
				symbols.result,
				symbols.e,
				symbols.prepared,
				symbols.blueprint,
			)

			// Check for statement.Exec error
			gosrc.WithIf("%s != nil", func(url.Values) error { return gosrc.Returns("-1", symbols.e) }, symbols.e)

			gosrc.Println("%s, %s := %s.RowsAffected()", symbols.count, symbols.e, symbols.result)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns("-1", symbols.e)
			}, symbols.e)

			return gosrc.Returns(symbols.count, writing.Nil)
		})

		if e == nil {
			record.registerImports("fmt")
			record.registerStoreMethod(writing.FuncDecl{
				Name:    methodName,
				Returns: returns,
				Params:  params,
			})
		}

		pw.CloseWithError(e)
	}()

	return pr
}
