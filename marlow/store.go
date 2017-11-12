package marlow

import "io"
import "net/url"
import "github.com/dadleyy/marlow/marlow/writing"

func writeStore(destination io.Writer, record marlowRecord) error {
	out := writing.NewGoWriter(destination)

	e := out.WithStruct(record.store(), func(url.Values) error {
		out.Println("*sql.DB")
		return nil
	})

	if e != nil {
		return e
	}

	record.registerImports("database/sql")
	return nil
}

// newStoreGenerator returns a reader that will generate the centralized record store for a given record.
func newStoreGenerator(record marlowRecord) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		e := writeStore(pw, record)
		pw.CloseWithError(e)
	}()

	return pr
}
