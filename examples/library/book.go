package library

type Book struct {
	ID        uint   `marlow:"column=id"`
	Title     string `marlow:"column=title"`
	AuthorID  uint   `marlow:"column=author_id&references=Author"`
	PageCount uint   `marlow:"column"`
}

func (b *Book) GetPageContents(page int) (string, error) {
	return "", nil
}
