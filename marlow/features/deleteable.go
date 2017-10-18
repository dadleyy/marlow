package features

import "io"
import "fmt"
import "net/url"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

// NewDeleteableGenerator is responsible for creating a generator that will write out the Delete api methods.
func NewDeleteableGenerator(record url.Values, fields map[string]url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()

	storeName := record.Get(constants.StoreNameConfigOption)
	recordName := record.Get(constants.RecordNameConfigOption)
	methodName := fmt.Sprintf("Delete%s", inflector.Pluralize(recordName))
	tableName := record.Get(constants.TableNameConfigOption)

	symbols := map[string]string{
		"ERROR":            "_error",
		"ROW_COUNT":        "_rowCount",
		"BLUEPRINT_PARAM":  "_blueprint",
		"STATEMENT_STRING": "_statement",
		"STATEMENT_RESULT": "_result",
	}

	params := []writing.FuncParam{
		{Type: fmt.Sprintf("*%s", blueprint{record: record}.Name()), Symbol: symbols["BLUEPRINT_PARAM"]},
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
			}, symbols["BLUEPRINT_PARAM"], symbols["BLUEPRINT_PARAM"])

			deleteString := fmt.Sprintf("DELETE FROM %s", tableName)

			gosrc.Println(
				"%s := fmt.Sprintf(\"%s %%s\", %s)",
				symbols["STATEMENT_STRING"],
				deleteString,
				symbols["BLUEPRINT_PARAM"],
			)

			gosrc.Println(
				"%s, %s := %s.e(%s + \";\")",
				symbols["STATEMENT_RESULT"],
				symbols["ERROR"],
				scope.Get("receiver"),
				symbols["STATEMENT_STRING"],
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols["ERROR"])
				return nil
			}, symbols["ERROR"])

			gosrc.Println("%s, %s := %s.RowsAffected()", symbols["ROW_COUNT"], symbols["ERROR"], symbols["STATEMENT_RESULT"])

			gosrc.WithIf("%s != nil", func(url.Values) error {
				gosrc.Println("return -1, %s", symbols["ERROR"])
				return nil
			}, symbols["ERROR"])

			gosrc.Println("return %s, nil", symbols["ROW_COUNT"])

			return nil
		})

		if e == nil {
			imports <- "fmt"
		}

		pw.CloseWithError(e)
	}()

	return pr
}
