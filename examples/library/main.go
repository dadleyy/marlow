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

func main() {
	os.Remove(dbFile)

	db, err := sql.Open("sqlite3", dbFile)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	sqlStmt := `
	create table authors (id integer not null primary key, name text);
	delete from authors;
	`

	_, err = db.Exec(sqlStmt)

	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	tx, err := db.Begin()

	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("insert into authors(id, name) values(?, ?)")

	if err != nil {
		log.Fatal(err)
	}

	defer stmt.Close()

	for i := 0; i < 10; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("author-%03d", i))

		if err != nil {
			log.Fatal(err)
		}
	}

	tx.Commit()

	store := models.AuthorStore{DB: db}

	a, e := store.FindAuthors(&models.AuthorQuery{})

	if e != nil {
		log.Fatalf("error file finding authors: %s", e.Error())
	}

	for _, author := range a {
		log.Printf("found author name[%s]", author.Name)
	}
}
