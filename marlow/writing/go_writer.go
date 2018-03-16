package writing

import "net/url"

// FuncDecl provides the structure of a golang function definition. ast.FuncDecl was too complex.
type FuncDecl struct {
	Name    string
	Params  []FuncParam
	Returns []string
}

// FuncParam represents the golang function parameter syntax
type FuncParam struct {
	Type   string
	Symbol string
}

// Block is a function callback that is used while generating golang control flow blocks.
type Block func(url.Values) error

// GoWriter defines an interface with a simple api for writing go code.
type GoWriter interface {
	WritePackage(string)
	WriteImport(string)
	WriteCall(...string) error
	WithFunc(string, []FuncParam, []string, Block) error
	WithMethod(string, string, []FuncParam, []string, Block) error
	WithIf(string, Block, ...interface{}) error
	WithIter(string, Block, ...interface{}) error
	WithStruct(string, Block) error
	WithInterface(string, Block) error
	Returns(...string) error
	Println(string, ...interface{}) error
	Comment(string, ...interface{})
}
