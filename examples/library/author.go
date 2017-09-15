package library
{


}0-0]-00-o0=1232e3123

type Book struct {
	Title     string `marlow:"column=title"`
	AuthorID  uint   `marlow:"column=author_id&references=Author"`
	PageCount uint   `marlow:"column"`
}

func (b *Book) GetPageContents(page int) (string, error) {
	return "", nil
}

type Author struct {
	Name string `marlow:"column=name"`
}
