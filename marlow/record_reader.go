package marlow

import "io"
import "fmt"
import "bytes"
import "go/ast"
import "reflect"
import "net/url"
import "strings"
import "github.com/gedex/inflector"
import "github.com/dadleyy/marlow/marlow/features"

func newRecordReader(root ast.Decl, imports chan<- string) (io.Reader, bool) {
	decl, ok := root.(*ast.GenDecl)

	if !ok {
		return nil, false
	}

	typeDecl, ok := decl.Specs[0].(*ast.TypeSpec)

	if !ok {
		return nil, false
	}

	structType, ok := typeDecl.Type.(*ast.StructType)

	if !ok {
		return nil, false
	}

	typeName := typeDecl.Name.String()

	recordConfig, recordFields := make(url.Values), make(map[string]url.Values)

	for _, f := range structType.Fields.List {
		if f.Tag == nil {
			continue
		}

		tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
		fieldConfig, e := url.ParseQuery(tag.Get("marlow"))

		if e != nil || len(f.Names) == 0 {
			continue
		}

		name := f.Names[0].String()

		if name == "table" || name == "_" {
			recordConfig = fieldConfig
			continue
		}

		fieldConfig.Set("type", fmt.Sprintf("%v", f.Type))
		recordFields[name] = fieldConfig
	}

	// Typically the generate will want to generate the API based on the name of the type, but allow override.
	if recordConfig.Get("recordName") == "" {
		recordConfig.Set("recordName", typeName)
	}

	if recordConfig.Get("table") == "" {
		name := recordConfig.Get("recordName")
		tableName := strings.ToLower(inflector.Pluralize(name))
		recordConfig.Set("table", tableName)
	}

	if recordConfig.Get("defaultLimit") == "" {
		recordConfig.Set("defaultLimit", "10")
	}

	if recordConfig.Get("storeName") == "" {
		name := recordConfig.Get("recordName")
		storeName := fmt.Sprintf("%sStore", name)
		recordConfig.Set("storeName", storeName)
	}

	pr, pw := io.Pipe()

	go func() {
		e := readRecord(pw, recordConfig, recordFields, imports)
		pw.CloseWithError(e)
	}()

	return pr, true
}

func readRecord(writer io.Writer, config url.Values, fields map[string]url.Values, imports chan<- string) error {
	buffer := new(bytes.Buffer)
	readers, enabled := make([]io.Reader, 0), make(map[string]bool)

	for _, fieldConfig := range fields {
		queryable := fieldConfig.Get("queryable")

		if _, e := enabled["queryable"]; queryable != "false" && !e {
			generator := features.NewQueryableGenerator(config, fields, imports)
			readers = append(readers, generator)
			enabled["queryable"] = true
		}
	}

	if len(readers) == 0 {
		comment := strings.NewReader(fmt.Sprintf("// [marlow no-features]: %s\n", config.Get("recordName")))
		_, e := io.Copy(writer, comment)
		return e
	}

	// If we had any features enabled, we need to also generate the blue print API.
	readers = append(
		readers,
		features.NewStoreGenerator(config, imports),
		features.NewBlueprintGenerator(config, fields, imports),
	)

	// Iterate over all our collected features, copying them into the buffer
	if _, e := io.Copy(buffer, io.MultiReader(readers...)); e != nil {
		return e
	}

	_, e := io.Copy(writer, buffer)
	return e
}
