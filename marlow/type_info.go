package marlow

import "go/types"
import "github.com/dadleyy/marlow/marlow/constants"

// getTypeInfo returns a types.BasicInfo mask value based on the string provided.
func getTypeInfo(fieldType string) types.BasicInfo {
	var typeInfo types.BasicInfo

	for _, t := range constants.NumericCustomTypes {
		if t == fieldType {
			return types.IsNumeric
		}
	}

	for _, t := range types.Typ {
		n := t.Name()

		if fieldType != n {
			continue
		}

		typeInfo = t.Info()
	}

	return typeInfo
}
