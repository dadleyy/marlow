package writing

import "net/url"

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
	WithFunc(string, []FuncParam, []string, Block) error
	WithMethod(string, string, []FuncParam, []string, Block) error
	WithIf(string, Block, ...interface{}) error
	WithIter(string, Block, ...interface{}) error
	WithStruct(string, Block) error
	Println(string, ...interface{})
	Comment(string, ...interface{})
}
