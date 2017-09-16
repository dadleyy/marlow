package marlow

import "log"
import "fmt"
import "strings"

type b func(*Writer) error

type Writer struct {
	*log.Logger
}

func (w *Writer) formatReturns(returns []string) (returnList string) {
	switch {
	case len(returns) == 1:
		returnList = returns[0]
	case len(returns) > 1:
		returnList = fmt.Sprintf("(%s)", strings.Join(returns, ","))
	default:
		returnList = ""
	}

	return
}

func (w *Writer) formatArgList(args map[string]string) string {
	list := make([]string, 0, len(args))

	for name, typeDef := range args {
		list = append(list, fmt.Sprintf("%s %s", name, typeDef))
	}

	return strings.Join(list, ",")
}

func (w *Writer) withFunc(name string, args map[string]string, returns []string, block b) error {
	returnList := w.formatReturns(returns)
	argList := w.formatArgList(args)

	w.Printf("func %s(%s) %s {", name, argList, returnList)

	if e := block(w); e != nil {
		return e
	}

	w.Printf("}")
	w.Println()

	return nil
}

func (w *Writer) withMetod(name string, typeName string, args map[string]string, returns []string, block b) error {
	returnList := w.formatReturns(returns)
	argList := w.formatArgList(args)
	w.Printf("func (%s *%s) %s(%s) %s {", strings.ToLower(typeName)[0:1], typeName, name, argList, returnList)

	if e := block(w); e != nil {
		return e
	}

	w.Printf("}")
	w.Println()

	return nil
}

func (w *Writer) withStruct(name string, block b) error {
	w.Printf("type %s struct {", name)

	if e := block(w); e != nil {
		return e
	}

	w.Printf("}")
	w.Println()

	return nil
}
