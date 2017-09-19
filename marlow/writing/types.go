package writing

import "fmt"
import "strings"

func mapListToWrappedCommaList(list []string, wrapper string) string {
	copied := make([]string, len(list))

	for i, v := range list {
		copied[i] = fmt.Sprintf("%s%s%s", wrapper, v, wrapper)
	}

	return strings.Join(copied, ",")
}

type SingleQuotedStringList []string

func (l SingleQuotedStringList) String() string {
	return mapListToWrappedCommaList(l, "'")
}

type StringSliceLiteral []string

func (l StringSliceLiteral) String() string {
	doubles := mapListToWrappedCommaList(l, "\"")
	return fmt.Sprintf("[]string{%s}", doubles)
}
