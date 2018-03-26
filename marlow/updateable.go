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

type updateOperation struct {
	name      string
	operation string
	valueless bool
}

func updater(record marlowRecord, fieldConfig url.Values, op *updateOperation) io.Reader {
	pr, pw := io.Pipe()
	column := fieldConfig.Get(constants.ColumnConfigOption)

	if op == nil {
		prefix := record.config.Get(constants.UpdateFieldMethodPrefixConfigOption)
		method := fmt.Sprintf("%s%s%s", prefix, record.name(), fieldConfig.Get("FieldName"))
		op = &updateOperation{name: method, operation: "", valueless: false}
	}

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

	// Check to see if we have an operation, and if the operation requires a value (hacky). If no value is required then
	// we no longer need the first user-provided argument to the deletion method.
	if op.valueless {
		params = []writing.FuncParam{
			{Type: fmt.Sprintf("*%s", record.config.Get(constants.BlueprintNameConfigOption)), Symbol: symbols.blueprint},
		}
	}

	returns := []string{
		"int64",
		"error",
	}

	go func() {
		gosrc := writing.NewGoWriter(pw)
		gosrc.Comment("[marlow] updater method for %s", column)

		e := gosrc.WithMethod(op.name, record.store(), params, returns, func(scope url.Values) error {
			logwriter := logWriter{output: gosrc, receiver: scope.Get("receiver")}

			// Prepare a value count to keep track of the amount of dynamic components will be sent into the query.
			gosrc.Println("%s := 1", symbols.valueCount)

			// Add the blueprint value count to the query component count.
			gosrc.WithIf("%s != nil && len(%s.Values()) > 0", func(url.Values) error {
				return gosrc.Println("%s = len(%s.Values()) + 1", symbols.valueCount, symbols.blueprint)
			}, symbols.blueprint, symbols.blueprint)

			// If our operation requires a value, we to create a variable in the source that will represent the string
			// holder during the sql statement execution. For postgres this value is placement-aware, e.g: $1.
			if !op.valueless {
				switch record.dialect() {
				case "postgres":
					gosrc.Println("%s := fmt.Sprintf(\"$%%d\", %s)", symbols.targetValue, symbols.valueCount)
					break
				default:
					gosrc.Println("%s := \"?\"", symbols.targetValue)
				}
			}

			command := fmt.Sprintf("UPDATE %s SET %s = %%s", record.table(), column)

			// If we have an operation, use it here instead.
			if op.operation != "" {
				command = fmt.Sprintf("UPDATE %s SET %s = %s", record.table(), column, op.operation)
			}

			// Start the update template string with the basic SQL-dialect `UPDATE <table> SET <column> = ?` syntax.
			template := fmt.Sprintf("fmt.Sprintf(\"%s\", %s)", command, symbols.targetValue)

			// If no value was required in the operation string we can simplify the sql.
			if op.valueless {
				template = fmt.Sprintf("\"%s\"", command)
			}

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
			if record.dialect() != "postgres" && !op.valueless {
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
			if record.dialect() == "postgres" && !op.valueless {
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
			Name:    op.name,
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

	for name, config := range record.fields {
		column := config.Get(constants.ColumnConfigOption)
		up := updater(record, config, nil)
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
				updater(record, config, &updateOperation{
					name:      fmt.Sprintf("Add%s%s", record.name(), name),
					operation: fmt.Sprintf("%s | %%s", column),
				}),

				updater(record, config, &updateOperation{
					name:      fmt.Sprintf("Drop%s%s", record.name(), name),
					operation: fmt.Sprintf("%s & ~%%s", column),
				}),
			}

			readers = append(readers, bitwise...)
		}

		readers = append(readers, up)
	}

	return io.MultiReader(readers...)
}
