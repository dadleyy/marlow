package main

import "os"
import "fmt"
import "log"
import "flag"
import "bufio"
import "strconv"
import "net/url"
import _ "github.com/lib/pq"
import _ "github.com/mattn/go-sqlite3"
import "database/sql"
import "github.com/dadleyy/marlow/examples/library/data"
import "github.com/dadleyy/marlow/examples/library/models"

const (
	defaultSQLiteDBFile  = "./library.db"
	psqlConnectionString = "user=%s dbname=%s port=%s sslmode=disable password=%s"
)

type dbConfig struct {
	sqlite struct {
		filename string
	}
	postgres struct {
		username string
		database string
		port     string
		password string
	}
}

type connections struct {
	postgres *sql.DB
	sqlite   *sql.DB
}

func (c *connections) Close() {
	c.postgres.Close()
	c.sqlite.Close()
}

func seed(db *sql.DB, filename string) error {
	schema, e := data.Asset("data/sqlite.sql")

	if e != nil {
		return e
	}

	if _, e = db.Exec(string(schema)); e != nil {
		return e
	}

	return nil
}

func initializeDatabases(config dbConfig) (*connections, error) {
	if _, e := os.Stat(config.sqlite.filename); e != nil {
		os.Remove(config.sqlite.filename)
	}

	sqlite, e := sql.Open("sqlite3", config.sqlite.filename)

	if e != nil {
		return nil, e
	}

	if e := seed(sqlite, "data/sqlite.sql"); e != nil {
		return nil, e
	}

	constr := fmt.Sprintf(
		psqlConnectionString,
		config.postgres.username,
		config.postgres.database,
		config.postgres.port,
		config.postgres.password,
	)

	postgres, e := sql.Open("postgres", constr)

	if e != nil {
		return nil, e
	}

	if e := seed(postgres, "data/postgres.sql"); e != nil {
		return nil, e
	}

	return &connections{
		postgres: postgres,
		sqlite:   sqlite,
	}, nil
}

type action struct {
	prompts []string
	handler func([]string) error
}

type modelStores struct {
	books  models.BookStore
	genres models.GenreStore
}

type inputHandler struct {
	stores modelStores
}

func (i *inputHandler) createBook(input string) error {
	values, e := url.ParseQuery(input)

	if e != nil {
		return fmt.Errorf("'values' must be a valid url query string, e: %s", e.Error())
	}

	author, e := strconv.Atoi(values.Get("author"))

	if e != nil {
		return fmt.Errorf("invalid author id: %d (error: %s)", author, e.Error())
	}

	pages, e := strconv.Atoi(values.Get("page-count"))

	if e != nil {
		return fmt.Errorf("invalid page-count: %d (error %s)", pages, e.Error())
	}

	var series sql.NullInt64

	if e := series.Scan(values.Get("series")); e != nil && values.Get("series") != "" {
		return fmt.Errorf("invalid series id: %d", series)
	}

	b := models.Book{
		Title:     values.Get("title"),
		AuthorID:  author,
		SeriesID:  series,
		PageCount: pages,
	}

	id, e := i.stores.books.CreateBooks(b)

	if e != nil {
		return fmt.Errorf("unable to create book: %s", e.Error())
	}

	fmt.Printf("created book: %d\n", id)

	return nil
}

func (i *inputHandler) searchBooks(query string) error {
	blueprint := &models.BookBlueprint{
		TitleLike: []string{fmt.Sprintf("%%%s%%", query)},
	}

	books, e := i.stores.books.FindBooks(blueprint)

	if e != nil {
		return e
	}

	count, e := i.stores.books.CountBooks(blueprint)

	if e != nil {
		return e
	}

	fmt.Printf("found %d books: \n", count)

	for _, b := range books {
		fmt.Printf("- %d: %s\n", b.ID, b.Title)
	}

	return nil
}

func (i *inputHandler) search(answers []string) error {
	switch answers[1] {
	case "book":
		return i.searchBooks(answers[2])
	}

	return fmt.Errorf("invalid type: %s", answers[1])
}

func (i *inputHandler) delete(answers []string) error {
	return nil
}

func (i *inputHandler) create(answers []string) error {
	switch answers[1] {
	case "book":
		return i.createBook(answers[2])
	}

	return fmt.Errorf("invalid type: %s", answers[1])
}

func (i *inputHandler) update(answers []string) error {
	return nil
}

func main() {
	config := dbConfig{}

	flag.StringVar(&config.sqlite.filename, "sqlite-file", defaultSQLiteDBFile, "file for sqlite connections")
	flag.StringVar(&config.postgres.username, "psql-username", "postgres", "username for psql connection")
	flag.StringVar(&config.postgres.database, "psql-database", "marlow_test", "database name for psql connection")
	flag.StringVar(&config.postgres.password, "psql-password", "", "password for psql connection")
	flag.StringVar(&config.postgres.port, "psql-port", "5432", "post for psql connection")

	flag.Parse()

	dbs, e := initializeDatabases(config)

	if e != nil {
		log.Fatal(e)
	}

	defer dbs.Close()

	stores := modelStores{
		models.NewBookStore(dbs.sqlite, nil),
		models.NewGenreStore(dbs.postgres, nil),
	}

	bookCount, e := stores.books.CountBooks(&models.BookBlueprint{})

	if e != nil {
		log.Fatal(e)
	}

	genreCount, e := stores.genres.CountGenres(&models.GenreBlueprint{})

	if e != nil {
		log.Fatal(e)
	}

	log.Printf("%d initial books", bookCount)
	log.Printf("%d initial genres", genreCount)

	inputs := bufio.NewReader(os.Stdin)

	// Buffering the prompts channel so a single answer handler can push multiple new prompts on.
	prompts, answers := make(chan string, 10), make(chan string)

	handler := inputHandler{stores}

	actions := map[string]struct {
		prompts []string
		handler func([]string) error
	}{
		"search": {[]string{"type (book/genre/author)", "query"}, handler.search},
		"update": {[]string{"type (book/genre/author)", "id", "values"}, handler.update},
		"delete": {[]string{"type (book/genre/author)", "id"}, handler.delete},
		"create": {[]string{"type (book/genre/author)", "values"}, handler.create},
	}

	go func() {
		prompts <- "search/update/delete/create/exit"
		stack := make([]string, 0, 4)

		for answer := range answers {
			if answer == "exit" {
				close(prompts)
				break
			}

			// We're at the first prompt of an action sequence. we should get the list of next prompts and prepare them.
			if len(stack) == 0 {
				stack = append(stack, answer)
				next, ok := actions[answer]

				// If the user types an unknown action, reset the stack to the start.
				if !ok {
					stack = make([]string, 0, 4)
					prompts <- "search/update/delete/create/exit"
					continue
				}

				for _, p := range next.prompts {
					prompts <- p
				}

				continue
			}

			action := stack[0]
			stack = append(stack, answer)

			// We aren't done answering all of our promts, continue on.
			if len(stack) != len(actions[action].prompts)+1 {
				continue
			}

			if e := actions[action].handler(stack); e != nil {
				log.Printf("unable to %s: %s", action, e.Error())
			}

			stack = make([]string, 0, 4)
			prompts <- "search/update/delete/create/exit"
		}
	}()

	for prompt := range prompts {
		fmt.Printf("%s: ", prompt)
		line, _, e := inputs.ReadLine()

		if e != nil {
			break
		}

		answers <- string(line)
	}

	close(answers)

	log.Println("done")
}
