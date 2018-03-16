package models

import "fmt"
import "time"
import "database/sql"

//go:generate marlowc -input author.go

// Example file for testing.

// Author represents an author of a book.
type Author struct {
	table        bool          `marlow:"tableName=authors"`
	ID           int           `marlow:"column=system_id&autoIncrement=true"`
	Name         string        `marlow:"column=name"`
	UniversityID sql.NullInt64 `marlow:"column=university_id"`
	ReaderRating float64       `marlow:"column=rating"`
	AuthorFlags  uint8         `marlow:"column=flags"`
	Birthday     time.Time     `marlow:"column=birthday"`
}

func (a *Author) String() string {
	return fmt.Sprintf("%v (born %v)", a.Name, a.Birthday.Format(time.RFC1123))
}
