package main

import "os"
import "log"
import "fmt"
import _ "github.com/mattn/go-sqlite3"
import "database/sql"
import "github.com/dadleyy/marlow/examples/library/models"

const (
	dbFile = "./library.db"
)

func withTx(db *sql.DB, block func(*sql.Tx) error) error {
	tx, e := db.Begin()

	if e != nil {
		return e
	}

	if e := block(tx); e != nil {
		return e
	}

	tx.Commit()
	return nil
}

func addBooks(tx *sql.Tx) error {
	stmt, e := tx.Prepare("insert into books(id, title, author_id, page_count) values(?, ?, ?, ?)")

	if e != nil {
		log.Fatal(e)
	}

	defer stmt.Close()

	for i := 0; i < 50; i++ {
		_, e = stmt.Exec(i, fmt.Sprintf("book-%03d", i), i, i)

		if e != nil {
			return e
		}
	}

	return nil
}

func addAuthors(tx *sql.Tx) error {
	stmt, e := tx.Prepare("insert into authors(id, name) values(?, ?)")

	if e != nil {
		log.Fatal(e)
	}

	defer stmt.Close()

	for i := 0; i < 50; i++ {
		_, e = stmt.Exec(i, fmt.Sprintf("author-%03d", i))

		if e != nil {
			return e
		}
	}

	return nil
}

func main() {
	os.Remove(dbFile)
	defer os.Remove(dbFile)

	db, err := sql.Open("sqlite3", dbFile)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	sqlStmt := `
	create table authors (id integer not null primary key, name text);
	delete from authors;
	create table books (
		id integer not null primary key,
		title text,
		author_id integer not null,
		page_count integer not null
	);
	delete from books;
	`

	_, err = db.Exec(sqlStmt)

	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	if e := withTx(db, addAuthors); e != nil {
		log.Fatalf("unable to add authors: %s", e.Error())
	}

	if e := withTx(db, addBooks); e != nil {
		log.Fatalf("unable to add books: %s", e.Error())
	}

	log.Printf("author query w/o values: %v", &models.AuthorBlueprint{})

	log.Printf("author query w values: %v", &models.AuthorBlueprint{
		ID: []int{123, 456},
	})

	authorStore := models.AuthorStore{DB: db}

	a, e := authorStore.FindAuthors(&models.AuthorBlueprint{
		ID: []int{1, 2, 3},
	})

	if e != nil {
		log.Fatalf("error file finding authors: %s", e.Error())
	}

	for _, author := range a {
		log.Printf("found author name[%s]", author.Name)
	}

	bookStore := models.BookStore{DB: db}
	b, e := bookStore.FindBooks(&models.BookBlueprint{
		ID: []int{1, 2},
	})

	if e != nil {
		log.Fatalf("error file finding authors: %s", e.Error())
	}

	for _, book := range b {
		log.Printf("found book: %s", book.Title)
	}

	log.Println("done")
}
