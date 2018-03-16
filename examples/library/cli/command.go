package cli

import "github.com/dadleyy/marlow/examples/library/models"

type Command func(*models.Stores, []string) error
