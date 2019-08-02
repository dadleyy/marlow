package models

import "fmt"
import "database/sql"

//go:generate marlowc -input book.go

// Example file for testing.

// Book represents a book in the example application
type Book struct {
	table         string        `marlow:"defaultLimit=10"`
	ID            int           `marlow:"column=system_id&autoIncrement=true"`
	Title         string        `marlow:"column=title"`
	AuthorID      int           `marlow:"column=author"`
	SeriesID      sql.NullInt64 `marlow:"column=series"`
	YearPublished int           `marlow:"column=year_published" json:"year_published"`
}

// String returns the book with good info.
func (b *Book) String() string {
	return fmt.Sprintf("%s (published in %d)", b.Title, b.YearPublished)
}
