package cli

import "fmt"
import "github.com/dadleyy/marlow/examples/library/models"

// Browse (TODO) provides an interactive session with the underlying database tables + marlow stores.
func Browse(store *models.Stores, args []string) error {
	fmt.Printf("starting library browser...\n")
	return nil
}
