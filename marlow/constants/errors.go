package constants

const (
	// InvalidDeletionBlueprint returned from the delete api when the blueprint generates no where clause.
	InvalidDeletionBlueprint = "deletion blueprints must generate limiting clauses"

	// InvalidGeneratedCodeError is the message that is returned when marlow generates invalid code. Typically a problem
	// with marlow, not necessarily the source data.
	InvalidGeneratedCodeError = "Marlow was unable to generate valid golang code. " +
		"Typically this is a problem with the marlow compiler\nitself, not necessarily the source structs. " +
		"Consider opening a github issue and including the source\nstruct configuration at https://github.com/dadleyy/marlow"
)
