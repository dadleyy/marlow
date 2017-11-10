package marlow

import "go/ast"

type recordExhaust struct {
	imports           chan<- string
	methods           chan<- *ast.FuncDecl
	registeredImports map[string]bool
}

func (e *recordExhaust) RegisterStoreMethods(imports ...*ast.FuncDecl) {
}

func (e *recordExhaust) RegisterImports(imports ...string) {
	if e.registeredImports == nil {
		e.registeredImports = make(map[string]bool)
	}

	for _, i := range imports {
		_, dupe := e.registeredImports[i]

		if dupe {
			continue
		}

		e.registeredImports[i] = true
		e.imports <- i
	}
}
