package main

import "fmt"
import "github.com/dadleyy/marlow/examples/library/models"

func main() {
	m := models.Book{}
	a := models.Author{}

	store := models.AuthorStore{}

	store.FindAuthors(models.AuthorQuery{})

	fmt.Printf("%v %v\n", m, a)
}
