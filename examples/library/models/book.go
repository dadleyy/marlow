package models

//go:generate marlowc -input book.go

// Example file for testing.

// Book represents a book in the example application
type Book struct {
	table     string `marlow:"defaultLimit=100&table=@@"`
	ID        uint   `marlow:"column=id"`
	Title     string `marlow:"column=title"`
	AuthorID  uint   `marlow:"column=author_id&references=Author"`
	PageCount uint   `marlow:"column=page_count"`
}

// GetPageContents is a dummy no-op function
func (b *Book) GetPageContents(page int) (string, error) {
	return "", nil
}
