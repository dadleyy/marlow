package models

import "database/sql"

//go:generate marlowc -input book.go

// Example file for testing.

// Book represents a book in the example application
type Book struct {
	table     string        `marlow:"defaultLimit=10"`
	ID        int           `marlow:"column=id&autoIncrement=true"`
	Title     string        `marlow:"column=title"`
	AuthorID  int           `marlow:"column=author&references=Author"`
	SeriesID  sql.NullInt64 `marlow:"column=series"`
	PageCount int           `marlow:"column=page_count"`
}

// GetPageContents is a dummy no-op function
func (b *Book) GetPageContents(page int) (string, error) {
	return "", nil
}
