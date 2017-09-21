package models

//go:generate marlowc -input book.go

// Example file for testing.

// Book represents a book in the example application
type Book struct {
	table     string `marlow:"defaultLimit=100&table=@@"`
	ID        int    `marlow:"column=id"`
	Title     string `marlow:"column=title"`
	AuthorID  int    `marlow:"column=author_id&references=Author"`
	PageCount int    `marlow:"column=page_count"`
}

// GetPageContents is a dummy no-op function
func (b *Book) GetPageContents(page int) (string, error) {
	return "", nil
}
