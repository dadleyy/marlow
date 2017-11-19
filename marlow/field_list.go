package marlow

import "strings"

type field struct {
	name   string
	column string
}

type fieldList []field

func (l fieldList) Less(i, j int) bool {
	return strings.ToLower(l[i].name) < strings.ToLower(l[j].name)
}

func (l fieldList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l fieldList) Len() int {
	return len(l)
}
