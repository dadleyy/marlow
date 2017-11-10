package marlow

import "io"
import "net/url"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

func writeStore(destination io.Writer, record url.Values, imports chan<- string) error {
	out := writing.NewGoWriter(destination)
	storeName := record.Get(constants.StoreNameConfigOption)

	e := out.WithStruct(storeName, func(url.Values) error {
		out.Println("*sql.DB")
		return nil
	})

	if e != nil {
		return e
	}

	imports <- "database/sql"
	return nil
}

// NewStoreGenerator returns a reader that will generate the centralized record store for a given record.
func NewStoreGenerator(record url.Values, imports chan<- string) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		e := writeStore(pw, record, imports)
		pw.CloseWithError(e)
	}()

	return pr
}
