package main

import "os"
import "fmt"
import "flag"
import "bytes"
import "github.com/dadleyy/marlow/examples/library/cli"
import "github.com/dadleyy/marlow/examples/library/models"

func main() {
	config := models.DatabaseConfig{}

	flag.StringVar(&config.SQLite.Filename, "sqlite-filename", "library.db", "the sqlite filename")
	flag.StringVar(&config.Postgres.Database, "postgres-database", "marlow_test", "the postgres database name")
	flag.StringVar(&config.Postgres.Username, "postgres-username", "postgres", "the postgres username")
	flag.StringVar(&config.Postgres.Hostname, "postgres-hostname", "0.0.0.0", "the postgres host")
	flag.StringVar(&config.Postgres.Port, "postgres-port", "5432", "the postgres port")
	flag.StringVar(&config.Postgres.Password, "postgres-password", "", "the postgres password")
	flag.Parse()

	connections := &models.DatabaseConnections{
		Config: &config,
	}

	if e := connections.Initialize(); e != nil {
		fmt.Printf("unable to initialize databases: %s\n", e.Error())
		os.Exit(2)
	}

	defer connections.Close()

	if len(flag.Args()) < 1 {
		fmt.Printf("must provide additional command\n")
		os.Exit(2)
	}

	var cmd cli.Command

	switch flag.Args()[0] {
	case "import":
		cmd = cli.Import
	case "browse":
		cmd = cli.Browse
	}

	logdump := new(bytes.Buffer)
	stores := connections.Stores(logdump)

	if e := cmd(stores, flag.Args()[1:]); e != nil {
		fmt.Printf("error: %s\n", e.Error())
		os.Exit(2)
	}

	fmt.Printf("done, queries:\n%s\n", logdump.String())
}
