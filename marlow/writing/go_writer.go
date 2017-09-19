package writing

import "net/url"

type FuncParam struct {
	Type   string
	Symbol string
}

type Block func(url.Values) error

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
