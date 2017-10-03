package constants

const (
	// TableNameConfigOption lets the marlow compiler know which sql table to associate with the current struct.
	TableNameConfigOption = "tableName"

	// DefaultLimitConfigOption is the 'table' field config key used to determine the default limit used in lookups.
	DefaultLimitConfigOption = "defaultLimit"

	// RecordNameConfigOption is the key used on the special table field to determine the return value of everything.
	RecordNameConfigOption = "recordName"

	// StoreNameConfigOption is the 'table' field key who's value will be used as the name of the main store struct.
	StoreNameConfigOption = "storeName"

	// BlueprintRangeFieldSuffixConfigOption is the string that will be appended to fields on the blueprint used for
	// searching ranges on numerical field types.
	BlueprintRangeFieldSuffixConfigOption = "blueprintRangeFieldSuffix"

	// BlueprintLikeFieldSuffixConfigOption is the string that will be appened to string/text fields and used for LIKE
	// searching by the queryable interface.
	BlueprintLikeFieldSuffixConfigOption = "blueprintLikeFieldSuffix"

	// StoreFindMethodPrefixConfigOption determines the prefix used when adding the main find/lookup method to the store.
	StoreFindMethodPrefixConfigOption = "storeFindMethodPrefix"
)
