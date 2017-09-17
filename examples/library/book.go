package library

// Example file for testing.

// Book represents a book in the example application
type Book struct {
	ID        uint   `marlow:"column=id"`
	Title     string `marlow:"column=title"`
	AuthorID  uint   `marlow:"column=author_id&references=Author"`
	PageCount uint   `marlow:"column"`
}

// GetPageContents is a dummy no-op function
func (b *Book) GetPageContents(page int) (string, error) {
	return "", nil
}
