package marlow

import "net/url"
import "github.com/dadleyy/marlow/marlow/constants"

// marlowRecord structs represent both the field-level and record-level configuration options for gerating the marlow stores.
type marlowRecord struct {
	config         url.Values
	fields         map[string]url.Values
	importChannel  chan<- string
	importRegistry map[string]bool
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

func (r *marlowRecord) name() string {
	return r.config.Get(constants.RecordNameConfigOption)
}

func (r *marlowRecord) store() string {
	return r.config.Get(constants.StoreNameConfigOption)
}

func (r *marlowRecord) table() string {
	return r.config.Get(constants.TableNameConfigOption)
}

func (r *marlowRecord) blueprint() string {
	return r.config.Get(constants.BlueprintNameConfigOption)
}
