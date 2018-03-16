package marlow

import "fmt"
import "strings"
import "github.com/dadleyy/marlow/marlow/writing"
import "github.com/dadleyy/marlow/marlow/constants"

type logWriter struct {
	output   writing.GoWriter
	receiver string
}

func (w *logWriter) AddLog(values ...string) {
	if w.output == nil || w.receiver == "" || len(values) == 0 {
		return
	}

	receiver := fmt.Sprintf("%s.%s", w.receiver, constants.StoreLoggerField)
	format := make([]string, len(values))

	for i := range values {
		format[i] = "%v"
	}

	w.output.WriteCall(
		"fmt.Fprintf",
		receiver,
		fmt.Sprintf("\"%s %s\\n\"", constants.LoggerStatementPrefix, strings.Join(format, " | ")),
		strings.Join(values, ","),
	)
}
