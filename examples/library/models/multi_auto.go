package models

import "time"
import "github.com/lib/pq"

// MultiAuto represents a record w/ mutliple auto-increment directives on a postgres model.
type MultiAuto struct {
	table     bool        `marlow:"tableName=multi_auto&dialect=postgres&primaryKey=id&softDelete=DeletedAt"`
	ID        uint        `marlow:"column=id&autoIncrement=true"`
	Status    string      `marlow:"column=status&autoIncrement=true"`
	Name      string      `marlow:"column=name"`
	CreatedAt time.Time   `marlow:"column=created_at"`
	DeletedAt pq.NullTime `marlow:"column=deleted_at"`
}
