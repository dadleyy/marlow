package marlow

import "fmt"
import "io"
import "go/ast"
import "go/token"
import "go/parser"
import "io/ioutil"

func Copy(destination io.Writer, input io.Reader) error {
	fs := token.NewFileSet()
	data, e := ioutil.ReadAll(input)

	if e != nil {
		return e
	}

	parsed, e := parser.ParseFile(fs, "", data, parser.AllErrors)

	if e != nil {
		return e
	}

	for _, d := range parsed.Decls {
		decl, ok := d.(*ast.GenDecl)

		if !ok {
			continue
		}

		typeDecl, ok := decl.Specs[0].(*ast.TypeSpec)

		if !ok {
			continue
		}

		structType, ok := typeDecl.Type.(*ast.StructType)

		if !ok {
			continue
		}

		for _, f := range structType.Fields.List {
			fmt.Fprintf(destination, "found: %s\n", f.Tag.Value)
		}
	}

	_, e = io.Copy(destination, input)
	return e
}

func NewWriter(destination io.WriteCloser) io.WriteCloser {
	pipeIn, pipeOut := io.Pipe()
	done := make(chan struct{})

	go func() {
		defer close(done)
		e := Copy(destination, pipeIn)
		destination.Close()
		pipeOut.CloseWithError(e)
	}()

	w := &writer{
		PipeWriter: pipeOut,
		done:       done,
	}

	return w
}

type writer struct {
	*io.PipeWriter
	done chan struct{}
}

func (w *writer) Close() error {
	if e := w.PipeWriter.Close(); e != nil {
		return e
	}

	<-w.done
	return nil
}
