package marlow

import "io"

import "fmt"

// import "go/ast"
// import "net/url"
// import "strings"
// import "reflect"
import "go/token"
import "go/parser"

const (
	compilationHeader = "// Do not edit! Compiled with github.com/dadleyy/marlow"
)

func Copy(destination io.Writer, input io.Reader) error {
	pr, pw := io.Pipe()

	go func() {
		fs := token.NewFileSet()
		_, e := parser.ParseFile(fs, "", input, parser.AllErrors)

		if e != nil {
			pw.CloseWithError(fmt.Errorf("fucker"))
			return
		}

		pw.Close()
	}()

	_, e := io.Copy(destination, pr)

	if e != nil {
		panic(e)
	}

	return e
}

func NewWriter(destination io.Writer) io.WriteCloser {
	pr, pw := io.Pipe()
	done := make(chan struct{})

	// Writes into the returned writer will be sent into the reader, which will then subsequently send the data to
	// the destination writer that was provided, completing the compilation step.
	go func() {
		defer close(done)
		e := Copy(destination, pr)
		pr.CloseWithError(e)
	}()

	w := &writer{
		PipeWriter: pw,
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
