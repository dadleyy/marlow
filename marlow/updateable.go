package marlow

import "io"
import "fmt"
import "net/url"
import "go/types"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type updaterSymbols struct {
	valueParam      string
	blueprint       string
	queryString     string
	queryResult     string
	queryError      string
	statementResult string
	statementError  string
	rowCount        string
	rowError        string
	valueSlice      string
	valueCount      string
	targetValue     string
}

func updater(record marlowRecord, fieldConfig url.Values, methodName, op string) io.Reader {
	pr, pw := io.Pipe()
	column := fieldConfig.Get(constants.ColumnConfigOption)

	symbols := updaterSymbols{
		valueParam:      "_updates",
		blueprint:       "_blueprint",
		queryString:     "_queryString",
		queryResult:     "_queryResult",
		queryError:      "_queryError",
		statementResult: "_statement",
		statementError:  "_se",
		rowCount:        "_rowCount",
		rowError:        "_re",
		valueSlice:      "_values",
		valueCount:      "_valueCount",
		targetValue:     "_target",
	}

	params := []writing.FuncParam{
		{Type: fieldConfig.Get("type"), Symbol: symbols.valueParam},
		{Type: fmt.Sprintf("*%s", record.config.Get(constants.BlueprintNameConfigOption)), Symbol: symbols.blueprint},
	}

	if fieldConfig.Get("type") == "sql.NullInt64" {
		params[0].Type = fmt.Sprintf("*%s", fieldConfig.Get("type"))
	}

	returns := []string{
		"int64",
		"error",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)
		gosrc.Comment("[marlow] updater method for %s", column)

		e := gosrc.WithMethod(methodName, record.store(), params, returns, func(scope url.Values) error {
			logwriter := logWriter{output: gosrc, receiver: scope.Get("receiver")}

			// Prepare a value count to keep track of the amount of dynamic components will be sent into the query.
			gosrc.Println("%s := 1", symbols.valueCount)

			// Add the blueprint value count to the query component count.
			gosrc.WithIf("%s != nil && len(%s.Values()) > 0", func(url.Values) error {
				return gosrc.Println("%s = len(%s.Values()) + 1", symbols.valueCount, symbols.blueprint)
			}, symbols.blueprint, symbols.blueprint)

			switch record.dialect() {
			case "postgres":
				gosrc.Println("%s := fmt.Sprintf(\"$%%d\", %s)", symbols.targetValue, symbols.valueCount)
				break
			default:
				gosrc.Println("%s := \"?\"", symbols.targetValue)
			}

			command := fmt.Sprintf("UPDATE %s SET %s = %%s", record.table(), column)

			if op != "" {
				command = fmt.Sprintf("UPDATE %s SET %s = %s", record.table(), column, op)
			}

			// Start the update template string with the basic SQL-dialect `UPDATE <table> SET <column> = ?` syntax.
			template := fmt.Sprintf("fmt.Sprintf(\"%s\", %s)", command, symbols.targetValue)

			gosrc.Println("%s := bytes.NewBufferString(%s)", symbols.queryString, template)

			// Add our blueprint to the WHERE section of our update statement buffer if it is not nil.
			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Println("fmt.Fprintf(%s, \" %%s\", %s)", symbols.queryString, symbols.blueprint)
			}, symbols.blueprint)

			// Write the query execution statement.
			gosrc.Println(
				"%s, %s := %s.Prepare(%s.String() + \";\")",
				symbols.statementResult,
				symbols.statementError,
				scope.Get("receiver"),
				symbols.queryString,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns("-1", symbols.statementError)
			}, symbols.statementError)

			gosrc.Println("defer %s.Close()", symbols.statementResult)

			// Create an array of `interface` values that will be used during the `Exec` portion of our transaction.
			gosrc.Println("%s := make([]interface{}, 0, %s)", symbols.valueSlice, symbols.valueCount)

			// The postgres dialect uses numbered placeholder values. If the record is using anything other than that, the
			// placeholder for the target value should appear first in the set of values sent to Exec.
			if record.dialect() != "postgres" {
				gosrc.Println("%s = append(%s, %s)", symbols.valueSlice, symbols.valueSlice, symbols.valueParam)
			}

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Println(
					"%s = append(%s, %s.Values()...)",
					symbols.valueSlice,
					symbols.valueSlice,
					symbols.blueprint,
				)
			}, symbols.blueprint)

			// If we're postgres, add our value to the very end of our value slice.
			if record.dialect() == "postgres" {
				gosrc.Println("%s = append(%s, %s)", symbols.valueSlice, symbols.valueSlice, symbols.valueParam)
			}

			logwriter.AddLog(symbols.queryString, symbols.valueSlice)

			gosrc.Println("%s, %s := %s.Exec(%s...)",
				symbols.queryResult,
				symbols.queryError,
				symbols.statementResult,
				symbols.valueSlice,
			)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns("-1", symbols.queryError)
			}, symbols.queryError)

			gosrc.Println("%s, %s := %s.RowsAffected()", symbols.rowCount, symbols.rowError, symbols.queryResult)

			gosrc.WithIf("%s != nil", func(url.Values) error {
				return gosrc.Returns("-1", symbols.rowError)
			}, symbols.rowError)

			return gosrc.Returns(symbols.rowCount, writing.Nil)
		})

		if e != nil {
			pw.CloseWithError(e)
			return
		}

		record.registerImports("fmt", "bytes")
		record.registerStoreMethod(writing.FuncDecl{
			Name:    methodName,
			Params:  params,
			Returns: returns,
		})
		pw.CloseWithError(nil)
	}()

	return pr
}

// newUpdateableGenerator is responsible for generating updating store methods.
func newUpdateableGenerator(record marlowRecord) io.Reader {
	readers := make([]io.Reader, 0, len(record.fields))
	prefix := record.config.Get(constants.UpdateFieldMethodPrefixConfigOption)

	for name, config := range record.fields {
		column := config.Get(constants.ColumnConfigOption)
		method := fmt.Sprintf("%s%s%s", prefix, record.name(), name)
		up := updater(record, config, method, "")
		fieldType := getTypeInfo(config.Get("type"))

		if _, bit := config[constants.ColumnBitmaskOption]; bit {
			valid := (fieldType & (types.IsUnsigned | types.IsInteger)) == fieldType

			if !valid {
				e := fmt.Errorf("bitmask columns must be unsigned integers, %s has type \"%s\"", column, config.Get("type"))
				pr, pw := io.Pipe()
				pw.CloseWithError(e)
				return pr
			}

			bitwise := []io.Reader{
				updater(record, config, fmt.Sprintf("Add%s%s", record.name(), name), fmt.Sprintf("%s | %%s", column)),
				updater(record, config, fmt.Sprintf("Drop%s%s", record.name(), name), fmt.Sprintf("%s & ~%%s", column)),
			}

			readers = append(readers, bitwise...)
		}

		readers = append(readers, up)
	}

	return io.MultiReader(readers...)
}
