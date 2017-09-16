package marlow

import "log"
import "fmt"
import "strings"

type b func(*Writer) error

type Writer struct {
	*log.Logger
}

func (w *Writer) withFunc(name string, args map[string]string, returns []string, block b) error {
	argList := make([]string, 0, len(args))
	var returnList string

	switch {
	case len(returns) == 1:
		returnList = returns[0]
	case len(returns) > 1:
		returnList = fmt.Sprintf("(%s)", strings.Join(returns, ","))
	default:
		returnList = ""
	}

	for name, typeDef := range args {
		argList = append(argList, fmt.Sprintf("%s %s", name, typeDef))
	}

	w.Printf("func %s(%s) %s {", name, strings.Join(argList, ","), returnList)

	if e := block(w); e != nil {
		return e
	}

	w.Printf("}")

	return nil
}

func (w *Writer) withStruct(name string, block b) error {
	w.Printf("type %s struct {", name)

	if e := block(w); e != nil {
		return e
	}

	w.Printf("}")

	return nil
}
