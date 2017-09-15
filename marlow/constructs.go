package marlow

import "io"
import "fmt"
import "net/url"

func newQueryConstruct(recordName string, fields map[string]url.Values) *queryConstruct {
	reader, writer := io.Pipe()

	c := &queryConstruct{
		PipeReader: reader,
		recordName: recordName,
		fields:     fields,
	}

	go func() {
		writer.CloseWithError(fmt.Errorf("fudge"))
	}()

	return c
}

type queryConstruct struct {
	*io.PipeReader
	recordName string
	fields     map[string]url.Values
}
