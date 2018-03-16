package cli

import "os"
import "fmt"
import "sync"
import "encoding/json"
import "github.com/dadleyy/marlow/examples/library/models"

type importModelList struct {
	Books       []*models.Book   `json:"books"`
	Authors     []*models.Author `json:"authors"`
	Genres      []*models.Genre  `json:"genres"`
	BookAuthors []struct {
		Author string
		Book   string
	} `json:"book_authors"`
	GenreTaxonomy []struct {
		Child  string `json:"name"`
		Parent string `json:"parent"`
	} `json:"genre_taxonomy"`
}

type genreImportChild struct {
	child  models.Genre
	parent string
}

type importJSONSource struct {
	Imports importModelList `json:"imports"`
}

// Import is the used by the example app that uses a json file and the generate store interfaces to populate records.
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

	children := make(chan genreImportChild)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		pending := make([]genreImportChild, 0, len(source.Imports.Genres))

		for g := range children {
			pending = append(pending, g)
		}

		for _, g := range pending {
			matches, e := stores.Genres.SelectGenreIDs(&models.GenreBlueprint{Name: []string{g.parent}})

			if e != nil || len(matches) != 1 {
				fmt.Printf("unable to create genre %s, cant find parent %s (e %v)", g.child.Name, g.parent, e)
				continue
			}

			if e := g.child.ParentID.Scan(matches[0]); e != nil {
				fmt.Printf("unable to create genre %s (e %v)", g.child.Name, e)
				continue
			}

			fmt.Printf("importing %s... ", g.child.Name)
			id, e := stores.Genres.CreateGenres(g.child)

			if e != nil {
				fmt.Printf("unable to create genre %s (e %v)\n", g.child.Name, e)
				continue
			}

			fmt.Printf("%d\n", id)
		}

		wg.Done()
	}()

	for _, g := range source.Imports.Genres {
		parent := ""
		fmt.Printf("importing %s...", g)

		for _, t := range source.Imports.GenreTaxonomy {
			if t.Child != g.Name {
				continue
			}

			parent = t.Parent
			children <- genreImportChild{child: *g, parent: parent}
			break
		}

		if parent != "" {
			fmt.Printf("%s was child of %s, delaying creation\n", g.Name, parent)
			continue
		}

		id, e := stores.Genres.CreateGenres(*g)

		if e != nil {
			return fmt.Errorf("failed import on genre %s (e %v)", g, e)
		}

		fmt.Printf(" %d\n", id)
	}

	close(children)
	wg.Wait()

	for _, b := range source.Imports.Books {
		var authorName string

		for _, ba := range source.Imports.BookAuthors {
			if ba.Book == b.Title {
				authorName = ba.Author
			}
		}

		if authorName == "" {
			fmt.Printf("skipping book: \"%s\", no author found\n", b.Title)
			continue
		}

		aid, e := stores.Authors.SelectAuthorIDs(&models.AuthorBlueprint{
			Name: []string{authorName},
		})

		if e != nil || len(aid) != 1 {
			return fmt.Errorf("failed import on book author lookup - found %d (e %v)", len(aid), e)
		}

		fmt.Printf("creating book %s... ", b)

		b.AuthorID = aid[0]

		id, e := stores.Books.CreateBooks(*b)

		if e != nil {
			fmt.Println()
			return fmt.Errorf("failed import on book create (e %v)", e)
		}

		fmt.Printf("%d\n", id)
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
