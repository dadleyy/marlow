package models

// marlow:ignore

// Stores is a convenience struct designed to group the generated stores into a single place.
type Stores struct {
	Authors AuthorStore
	Books   BookStore
	Genres  GenreStore
}
