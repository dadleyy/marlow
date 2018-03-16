package cli

import "github.com/dadleyy/marlow/examples/library/models"

// Command represents a type that used by the example app cli.
type Command func(*models.Stores, []string) error
