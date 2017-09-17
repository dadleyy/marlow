package marlow

import "log"
import "fmt"
import "strings"
import "net/url"

type b func(url.Values) error

// Writer wraps the log.Logger interface with several handy functions for writing go code.
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

func (w *Writer) withBlock(start string, block b, v url.Values) error {
	w.Printf("%s {", start)

	if e := block(v); e != nil {
		return e
	}

	w.Printf("}")
	w.Println()
	return nil
}

func (w *Writer) withFunc(name string, args map[string]string, returns []string, block b) error {
	returnList := w.formatReturns(returns)
	argList := w.formatArgList(args)
	funcDef := fmt.Sprintf("func %s(%s) %s", name, argList, returnList)
	return w.withBlock(funcDef, block, make(url.Values))
}

func (w *Writer) withIter(condition string, block b) error {
	return w.withBlock(fmt.Sprintf("for %s", condition), block, make(url.Values))
}

func (w *Writer) withMetod(name string, typeName string, args map[string]string, returns []string, block b) error {
	returnList := w.formatReturns(returns)
	argList := w.formatArgList(args)
	receiver := strings.ToLower(typeName)[0:1]
	funcDef := fmt.Sprintf("func (%s *%s) %s(%s) %s", receiver, typeName, name, argList, returnList)
	c := make(url.Values)
	c.Set("receiver", receiver)
	return w.withBlock(funcDef, block, c)
}

func (w *Writer) withIf(condition string, block b) error {
	return w.withBlock(fmt.Sprintf("if %s", condition), block, make(url.Values))
}

func (w *Writer) withStruct(name string, block b) error {
	return w.withBlock(fmt.Sprintf("type %s struct", name), block, make(url.Values))
}
