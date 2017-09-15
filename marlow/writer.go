package marlow

import "io"
import "fmt"
import "go/ast"
import "net/url"
import "strings"
import "reflect"
import "go/token"
import "go/parser"

const (
	compilationHeader = "// Do not edit! Compiled with github.com/dadleyy/marlow"
)

func Copy(destination io.Writer, input io.Reader) error {
	pipeIn, pipeOut := io.Pipe()

	go func() {
		fs := token.NewFileSet()
		parsed, e := parser.ParseFile(fs, "", input, parser.AllErrors)

		if e != nil {
			pipeOut.CloseWithError(e)
			return
		}

		packageName := parsed.Name.String()

		fmt.Fprintln(pipeOut, compilationHeader)

		store := make(recordStore)

		// Iterate over every declaration in the parsed golang source file.
		for _, d := range parsed.Decls {
			decl, ok := d.(*ast.GenDecl)

			// Only deal with struct type declarations
			if !ok || decl.Tok != token.TYPE || len(decl.Specs) != 1 {
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

			typeName := typeDecl.Name.String()

			// Iterate over the struct fields, pulling out marlow tag information
			for _, f := range structType.Fields.List {
				tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
				config, err := url.ParseQuery(tag.Get("marlow"))

				if err != nil || len(f.Names) == 0 {
					continue
				}

				fieldName := f.Names[0].String()
				typeEntry, ok := store[typeName]

				if !ok {
					typeEntry = make(record)
					store[typeName] = typeEntry
				}

				config.Set("type", fmt.Sprintf("%v", f.Type))
				typeEntry[fieldName] = config
			}
		}

		compiler := NewGenerator(&store)

		fmt.Fprintf(pipeOut, "package %s\n\n", packageName)

		if _, e := io.Copy(pipeOut, compiler); e != nil {
			pipeOut.CloseWithError(e)
			return
		}

		pipeOut.Close()
	}()

	_, e := io.Copy(destination, pipeIn)
	return e
}

func NewWriter(destination io.Writer) io.WriteCloser {
	pipeIn, pipeOut := io.Pipe()
	done := make(chan struct{})

	// Writes into the returned writer will be sent into the reader, which will then subsequently send the data to
	// the destination writer that was provided, completing the compilation step.
	go func() {
		defer close(done)
		e := Copy(destination, pipeIn)
		pipeIn.CloseWithError(e)
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
