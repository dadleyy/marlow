package writing

import "fmt"
import "strings"

const (
	// EmptyString is an empty golang string
	EmptyString = "\"\""

	// Nil is nil
	Nil = "nil"
)

func mapListToWrappedCommaList(list []string, wrapper string) string {
	copied := make([]string, len(list))

	for i, v := range list {
		copied[i] = fmt.Sprintf("%s%s%s", wrapper, v, wrapper)
	}

	return strings.Join(copied, ",")
}

// SingleQuotedStringList will return itself as a comma delimited list of quoted strings when stringified.
type SingleQuotedStringList []string

func (l SingleQuotedStringList) String() string {
	return mapListToWrappedCommaList(l, "'")
}

// StringSliceLiteral will return itself as the golang literal syntax for string slices.
type StringSliceLiteral []string

func (l StringSliceLiteral) String() string {
	doubles := mapListToWrappedCommaList(l, "\"")
	return fmt.Sprintf("[]string{%s}", doubles)
}
