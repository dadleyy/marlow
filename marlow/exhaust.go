package marlow

import "go/ast"

// FeatureExhaust defines an interface that is used by generators to comunicate dependencies during their code gen.
type FeatureExhaust interface {
	RegisterImports(...string)
	RegisterStoreMethods(...*ast.FuncDecl)
}
