package constants

const (
	// StoreLoggerField is the internal field on stores for the io.Writer log stream
	StoreLoggerField = "logger"

	// PrimaryKeyColumnConfigOption specifies the primary key on the record
	PrimaryKeyColumnConfigOption = "primaryKey"

	// DialectConfigOption determines which dialect to use when building queries via blueprint.
	DialectConfigOption = "dialect"

	// TableNameConfigOption lets the marlow compiler know which sql table to associate with the current struct.
	TableNameConfigOption = "tableName"

	// DefaultLimitConfigOption is the 'table' field config key used to determine the default limit used in lookups.
	DefaultLimitConfigOption = "defaultLimit"

	// RecordSoftDeleteConfigOption is the config option that points to the field responsible for storing deleted-ness.
	RecordSoftDeleteConfigOption = "softDelete"

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

	// BlueprintNameSuffix is added after the record name for the type that can be stringifyed into valid sql code.
	BlueprintNameSuffix = "Blueprint"

	// BlueprintNameConfigOption holds the blueprint name on the record config.
	BlueprintNameConfigOption = "blueprintName"

	// StoreFindMethodPrefixConfigOption determines the prefix used when adding the main find/lookup method to the store.
	StoreFindMethodPrefixConfigOption = "storeFindMethodPrefix"

	// StoreCountMethodPrefixConfigOption determines the prefix used when adding the main count method to the store.
	StoreCountMethodPrefixConfigOption = "storeCountMethodPrefix"

	// StoreSelectMethodPrefixConfigOption determines the prefix used when adding the single field select methods.
	StoreSelectMethodPrefixConfigOption = "storeSelectMethodPrefix"

	// ColumnAutoIncrementFlag used to determine if primary key should be inserted during creation.
	ColumnAutoIncrementFlag = "autoIncrement"

	// UpdateFieldMethodPrefixConfigOption is the method prefix of update methods for individual fields.
	UpdateFieldMethodPrefixConfigOption = "updateMethodPrefix"

	// ColumnConfigOption is the key of the value used on individual fields that represents which column marlow queries.
	ColumnConfigOption = "column"

	// ColumnBitmaskOption is used to indicate a field is a bitmask & can be used to generate bitwise ops.
	ColumnBitmaskOption = "bitmask"

	// QueryableConfigOption boolean value, true/false based on fields ability to be updated.
	QueryableConfigOption = "queryable"

	// UpdateableConfigOption boolean value, true/false based on fields ability to be updated.
	UpdateableConfigOption = "updateable"

	// DeleteableConfigOption boolean record config option for generating the deletion api methods.
	DeleteableConfigOption = "deletable"

	// CreateableConfigOption boolean record config option for generating the creation api methods.
	CreateableConfigOption = "createable"
)
