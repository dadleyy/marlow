package marlow

import "go/ast"
import "net/url"

// marlowRecord structs represent both the field-level and record-level configuration options for gerating the marlow stores.
type marlowRecord struct {
	config  url.Values
	fields  map[string]url.Values
	imports chan<- string
	store   chan<- *ast.FuncDecl
}
