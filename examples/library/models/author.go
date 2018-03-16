package models

import "fmt"
import "time"
import "database/sql"

//go:generate marlowc -input author.go

// Example file for testing.

const (
	// AuthorImported indicates the author was imported from an external source.
	AuthorImported = 1 << iota

	// AuthorHasMultipleTitles is a convenience value indicating the author has multiple titles.
	AuthorHasMultipleTitles
)

// Author represents an author of a book.
type Author struct {
	table        bool          `marlow:"tableName=authors"`
	ID           int           `marlow:"column=system_id&autoIncrement=true"`
	Name         string        `marlow:"column=name"`
	UniversityID sql.NullInt64 `marlow:"column=university_id"`
	ReaderRating float64       `marlow:"column=rating"`
	AuthorFlags  uint8         `marlow:"column=flags&bitmask"`
	Birthday     time.Time     `marlow:"column=birthday"`
}

func (a *Author) String() string {
	return fmt.Sprintf("%v (born %v)", a.Name, a.Birthday.Format(time.RFC1123))
}
