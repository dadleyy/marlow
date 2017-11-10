package marlow

import "go/ast"

type testExhaust struct {
	imports map[string]bool
	methods map[string]bool
}

func (e *testExhaust) RegisterStoreMethods(methods ...*ast.FuncDecl) {
	if e.methods == nil {
		e.methods = make(map[string]bool)
	}

	for _, k := range methods {
		e.methods[k.Name.Name] = true
	}
}

func (e *testExhaust) RegisterImports(imports ...string) {
	if e.imports == nil {
		e.imports = make(map[string]bool)
	}

	for _, k := range imports {
		e.imports[k] = true
	}
}
