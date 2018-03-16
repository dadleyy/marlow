package cli

import "os"
import "fmt"
import "encoding/json"
import "github.com/dadleyy/marlow/examples/library/models"

type importModelList struct {
	Books   []*models.Book   `json:"books"`
	Authors []*models.Author `json:"authors"`
	Genres  []*models.Genre  `json:"genres"`
}

type importJSONSource struct {
	Imports importModelList `json:"imports"`
}

func Import(stores *models.Stores, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("must provide a filename")
	}

	filename := args[0]

	if s, e := os.Stat(filename); e != nil || s.IsDir() {
		return fmt.Errorf("filename must exist and be a regular file (given %s)", filename)
	}

	file, e := os.Open(filename)

	if e != nil {
		return fmt.Errorf("unable to open import file (e %v)", e)
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	var source importJSONSource

	if e := decoder.Decode(&source); e != nil {
		return fmt.Errorf("unable to decode json (e %v)", e)
	}

	for _, a := range source.Imports.Authors {
		fmt.Printf("importing %s...", a)
		id, e := stores.Authors.CreateAuthors(*a)

		if e != nil {
			fmt.Println()
			return fmt.Errorf("failed import on %s (e %v)", a, e)
		}

		fmt.Printf(" %d\n", id)
	}

	for _, g := range source.Imports.Genres {
		fmt.Printf("importing %s...", g)

		id, e := stores.Genres.CreateGenres(*g)

		if e != nil {
			return fmt.Errorf("failed import on genre %s (e %v)", g, e)
		}

		fmt.Printf(" %d\n", id)
	}

	counts := struct {
		authors int
		genres  int
	}{}

	counts.authors, e = stores.Authors.CountAuthors(nil)

	if e != nil {
		return fmt.Errorf("unable to get import summary (e %v)", e)
	}

	counts.genres, e = stores.Genres.CountGenres(nil)

	if e != nil {
		return fmt.Errorf("unable to get import summary (e %v)", e)
	}

	fmt.Println(fmt.Sprintf("import summary: %d authors, %d genres", counts.authors, counts.genres))
	return nil
}
