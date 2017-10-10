package models

import "database/sql"

//go:generate marlowc -input author.go

// Example file for testing.

// Author represents an author of a book.
type Author struct {
	table        bool          `marlow:"tableName=authors"`
	ID           int           `marlow:"column=id"`
	Name         string        `marlow:"column=name"`
	UniversityID sql.NullInt64 `marlow:"column=university_id"`
}
