package marlow

import "io"
import "fmt"
import "bytes"
import "sync"
import "go/ast"
import "regexp"
import "reflect"
import "net/url"
import "strings"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

const (
	// DefaultBlueprintLimit is the default limit that will be used in blueprints unless one is configured on the record.
	DefaultBlueprintLimit = 100
)

var nameValidationRegex = regexp.MustCompile("^[A-z_]+$")

func newRecordConfig(typeName string) url.Values {
	config := make(url.Values)
	config.Set(constants.RecordNameConfigOption, typeName)
	tableName := strings.ToLower(inflector.Pluralize(typeName))
	config.Set(constants.TableNameConfigOption, tableName)
	config.Set(constants.DefaultLimitConfigOption, fmt.Sprintf("%d", DefaultBlueprintLimit))
	storeName := fmt.Sprintf("%sStore", typeName)
	blueprintName := fmt.Sprintf("%s%s", typeName, constants.BlueprintNameSuffix)
	config.Set(constants.StoreNameConfigOption, storeName)

	config.Set(constants.BlueprintNameConfigOption, blueprintName)
	config.Set(constants.BlueprintRangeFieldSuffixConfigOption, "Range")
	config.Set(constants.BlueprintLikeFieldSuffixConfigOption, "Like")

	config.Set(constants.StoreFindMethodPrefixConfigOption, "Find")
	config.Set(constants.StoreCountMethodPrefixConfigOption, "Count")
	config.Set(constants.UpdateFieldMethodPrefixConfigOption, "Update")
	config.Set(constants.StoreSelectMethodPrefixConfigOption, "Select")
	return config
}

func parseFieldType(config *url.Values, f *ast.Field) error {
	if f == nil || f.Names == nil || len(f.Names) != 1 {
		return fmt.Errorf("invalid field: %v", f)
	}

	name := f.Names[0]

	// Convert our field's type to it's string counterpart.
	fieldType := fmt.Sprintf("%v", f.Type)

	// Error on slice types
	if _, ok := f.Type.(*ast.ArrayType); ok == true {
		return fmt.Errorf("slice types not supported by marlow, field: %s", name)
	}

	expr := f.Type

	if ptr, ok := f.Type.(*ast.StarExpr); ok {
		expr = ptr.X
	}

	// Check to see if this field is a complex type - one that refers to an exported type from another package.
	selector, ok := expr.(*ast.SelectorExpr)

	// If the field is a complex type, make an note of the import that it is referring to - this will be mapped to the
	// original import path from the source package by our import processor.
	if ok {
		fieldType = fmt.Sprintf("%s.%s", selector.X, selector.Sel)
		config.Set("import", fmt.Sprintf("%s", selector.X))
	}

	config.Set("type", fieldType)
	return nil
}

func parseField(f *ast.Field) (string, url.Values, bool) {
	if f == nil || f.Names == nil || f.Tag == nil {
		return "", nil, false
	}

	tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
	config, e := url.ParseQuery(tag.Get("marlow"))

	if e != nil || len(f.Names) == 0 {
		return "", nil, false
	}

	if len(f.Names) != 1 {
		return "", nil, false
	}

	name := f.Names[0].String()

	config.Set("FieldName", name)

	return name, config, true
}

func parseStruct(d ast.Decl) (*ast.StructType, string, bool) {
	decl, ok := d.(*ast.GenDecl)

	if !ok {
		return nil, "", false
	}

	typeDecl, ok := decl.Specs[0].(*ast.TypeSpec)

	if !ok {
		return nil, "", false
	}

	structType, ok := typeDecl.Type.(*ast.StructType)

	if !ok {
		return nil, "", false
	}

	typeName := typeDecl.Name.String()
	return structType, typeName, true
}

func newRecordReader(root ast.Decl, imports chan<- string) (io.Reader, bool) {
	structType, typeName, ok := parseStruct(root)

	if !ok {
		return nil, false
	}

	recordConfig, recordFields := newRecordConfig(typeName), make(map[string]url.Values)

	columnMap := make(map[string]string)

	pr, pw := io.Pipe()

	for _, f := range structType.Fields.List {
		name, fieldConfig, ok := parseField(f)

		if !ok {
			continue
		}

		if name == "table" || name == "_" {
			for k := range fieldConfig {
				v := fieldConfig.Get(k)
				recordConfig.Set(k, v)
			}

			continue
		}

		columnName := fieldConfig.Get(constants.ColumnConfigOption)

		// If the column config option is the dash, skip any marlow related generation for it.
		if columnName == "-" {
			continue
		}

		// If the column name is empty, use the lowercased field name as the value.
		if columnName == "" {
			columnName = strings.ToLower(name)
			fieldConfig.Set(constants.ColumnConfigOption, columnName)
		}

		if otherField, dupe := columnMap[columnName]; dupe == true {
			pw.CloseWithError(fmt.Errorf("duplicate column \"%s\" for fields: %s & %s", columnName, otherField, name))
			return pr, true
		}

		columnMap[columnName] = name

		if nameValidationRegex.MatchString(columnName) != true {
			pw.CloseWithError(fmt.Errorf("invalid column name for %s: %s", name, columnName))
			return pr, true
		}

		if e := parseFieldType(&fieldConfig, f); e != nil {
			pw.CloseWithError(e)
			return pr, true
		}

		recordFields[name] = fieldConfig
	}

	if nameValidationRegex.MatchString(recordConfig.Get(constants.TableNameConfigOption)) != true {
		pw.CloseWithError(fmt.Errorf("invalid-table"))
		return pr, true
	}

	go func() {
		record := marlowRecord{
			config:        recordConfig,
			fields:        recordFields,
			importChannel: imports,
			storeChannel:  make(chan writing.FuncDecl),
		}

		e := readRecord(pw, record)
		pw.CloseWithError(e)
	}()

	return pr, true
}

func readRecord(writer io.Writer, record marlowRecord) error {
	buffer := new(bytes.Buffer)

	readers := make([]io.Reader, 0, 4)

	features := map[string]func(marlowRecord) io.Reader{
		constants.CreateableConfigOption: newCreateableGenerator,
		constants.UpdateableConfigOption: newUpdateableGenerator,
		constants.DeleteableConfigOption: newDeleteableGenerator,
		constants.QueryableConfigOption:  newQueryableGenerator,
	}

	for flag, generator := range features {
		v := record.config.Get(flag)

		if v == "false" {
			continue
		}

		g := generator(record)
		readers = append(readers, g)
	}

	if len(readers) == 0 {
		comment := strings.NewReader(
			fmt.Sprintf("/* [marlow no-features]: %s */\n\n", record.config.Get(constants.RecordNameConfigOption)),
		)

		_, e := io.Copy(writer, comment)
		return e
	}

	// If we had any features enabled, we need to also generate the blue print API.
	readers = append(readers, newBlueprintGenerator(record))

	methods := make(map[string]writing.FuncDecl)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		for method := range record.storeChannel {
			if _, d := methods[method.Name]; d {
				continue
			}

			methods[method.Name] = method
		}
		wg.Done()
	}()

	// Iterate over all our collected features, copying them into the buffer
	if _, e := io.Copy(buffer, io.MultiReader(readers...)); e != nil {
		return e
	}

	close(record.storeChannel)
	wg.Wait()

	store := newStoreGenerator(record, methods)
	_, e := io.Copy(writer, io.MultiReader(buffer, store))
	return e
}
