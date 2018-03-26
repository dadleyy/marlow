package marlow

import "fmt"
import "sort"
import "strings"
import "net/url"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

// marlowRecord structs represent both the field-level and record-level configuration options for gerating the marlow stores.
type marlowRecord struct {
	config url.Values
	fields map[string]url.Values

	importChannel  chan<- string
	importRegistry map[string]bool

	storeChannel chan writing.FuncDecl
}

func (r *marlowRecord) fieldList(filter func(url.Values) bool) fieldList {
	list := make(fieldList, 0, len(r.fields))

	for name, c := range r.fields {
		column := fmt.Sprintf("%s.%s", r.table(), c.Get(constants.ColumnConfigOption))

		if filter != nil && filter(c) != true {
			continue
		}

		list = append(list, field{name: name, column: column})
	}

	sort.Sort(list)

	return list
}

func (r *marlowRecord) deletionField() url.Values {
	target := r.config.Get(constants.RecordSoftDeleteConfigOption)

	if target == "" {
		return nil
	}

	return r.fields[target]
}

func (r *marlowRecord) registerStoreMethod(method writing.FuncDecl) {
	r.storeChannel <- method
}

func (r *marlowRecord) registerImports(imports ...string) {
	registry := r.importRegistry

	if registry == nil {
		r.importRegistry = make(map[string]bool)
		registry = r.importRegistry
	}

	for _, name := range imports {
		_, dupe := registry[name]

		if dupe {
			continue
		}

		r.importChannel <- name
		registry[name] = true
	}
}

func (r *marlowRecord) primaryKeyColumn() string {
	if r == nil {
		return ""
	}

	if recordValue := r.config.Get(constants.PrimaryKeyColumnConfigOption); recordValue != "" {
		return recordValue
	}

	for _, config := range r.fields {
		if v := config.Get(constants.PrimaryKeyColumnConfigOption); v == "true" {
			return config.Get(constants.ColumnConfigOption)
		}
	}

	return ""
}

func (r *marlowRecord) external() string {
	return r.config.Get(constants.StoreNameConfigOption)
}

func (r *marlowRecord) name() string {
	return r.config.Get(constants.RecordNameConfigOption)
}

func (r *marlowRecord) dialect() string {
	return r.config.Get(constants.DialectConfigOption)
}

func (r *marlowRecord) store() string {
	storeName := r.external()

	if storeName == "" {
		return ""
	}

	return strings.ToLower(storeName[0:1]) + storeName[1:]
}

func (r *marlowRecord) table() string {
	return r.config.Get(constants.TableNameConfigOption)
}

func (r *marlowRecord) blueprint() string {
	return r.config.Get(constants.BlueprintNameConfigOption)
}
