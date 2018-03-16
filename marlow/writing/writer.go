package writing

import "io"
import "log"
import "fmt"
import "strings"
import "net/url"

// NewGoWriter returns an instance of the GoWriter interface which is useful for writing golang code.
func NewGoWriter(destination io.Writer) GoWriter {
	return &goWriter{
		Logger: log.New(destination, "", 0),
	}
}

type goWriter struct {
	*log.Logger
}

func (w *goWriter) WriteCall(parts ...string) error {
	if len(parts) == 0 {
		return nil
	}

	formatted := fmt.Sprintf("%s()", parts[0])

	if len(parts) > 1 {
		formatted = fmt.Sprintf("%s(%s)", parts[0], strings.Join(parts[1:], ","))
	}

	w.Logger.Printf("%s\n", formatted)
	return nil
}

func (w *goWriter) Println(msg string, extras ...interface{}) error {
	formatted := fmt.Sprintf(msg, extras...)
	w.Logger.Printf("%s\n", formatted)
	return nil
}

func (w *goWriter) Returns(msg ...string) error {
	return w.Println("return %s", strings.Join(msg, ","))
}

func (w *goWriter) Comment(msg string, keys ...interface{}) {
	comment := fmt.Sprintf(msg, keys...)
	w.Println(fmt.Sprintf("// %s", comment))
}

func (w *goWriter) WritePackage(packageName string) {
	w.Printf("package %s\n", packageName)
}

func (w *goWriter) WriteImport(importName string) {
	w.Printf("import \"%s\"\n", importName)
}

func (w *goWriter) WithFunc(name string, args []FuncParam, returns []string, block Block) error {
	returnList := w.formatReturns(returns)
	argList := w.formatArgList(args)
	funcDef := fmt.Sprintf("func %s(%s) %s", name, argList, returnList)
	return w.withBlock(funcDef, block, make(url.Values))
}

func (w *goWriter) WithIter(condition string, block Block, symbols ...interface{}) error {
	if len(condition) == 0 {
		return fmt.Errorf("invalid-condition")
	}

	formattedCondition := fmt.Sprintf(condition, symbols...)

	return w.withBlock(fmt.Sprintf("for %s", formattedCondition), block, make(url.Values))
}

func (w *goWriter) WithMethod(name string, typeName string, args []FuncParam, returns []string, block Block) error {
	returnList := w.formatReturns(returns)
	argList := w.formatArgList(args)

	if len(typeName) >= 1 != true {
		return fmt.Errorf("invalid-receiver")
	}

	receiver := strings.ToLower(typeName)[0:1]
	funcDef := fmt.Sprintf("func (%s *%s) %s(%s) %s", receiver, typeName, name, argList, returnList)
	c := make(url.Values)
	c.Set("receiver", receiver)
	return w.withBlock(funcDef, block, c)
}

func (w *goWriter) WithIf(condition string, block Block, symbols ...interface{}) error {
	if len(condition) == 0 {
		return fmt.Errorf("invalid-condition")
	}

	formattedCondition := fmt.Sprintf(condition, symbols...)

	return w.withBlock(fmt.Sprintf("if %s", formattedCondition), block, make(url.Values))
}

func (w *goWriter) WithInterface(name string, block Block) error {
	return w.withType(name, "interface", block)
}

func (w *goWriter) WithStruct(name string, block Block) error {
	return w.withType(name, "struct", block)
}

func (w *goWriter) withType(name string, typename string, block Block) error {
	if len(name) == 0 {
		return fmt.Errorf("invalid-name")
	}

	return w.withBlock(fmt.Sprintf("type %s %s", name, typename), block, make(url.Values))
}

func (w *goWriter) formatReturns(returns []string) (returnList string) {
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

func (w *goWriter) formatArgList(args []FuncParam) string {
	list := make([]string, 0, len(args))

	for _, def := range args {
		list = append(list, fmt.Sprintf("%s %s", def.Symbol, def.Type))
	}

	return strings.Join(list, ",")
}

func (w *goWriter) withBlock(start string, block Block, v url.Values) error {
	w.Printf("%s {", start)
	defer w.Logger.Printf("}\n\n")

	if block == nil {
		return nil
	}

	return block(v)
}
