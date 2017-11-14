package models

import "database/sql"

type Genre struct {
	table    bool          `marlow:"tableName=genres&dialect=postgres&primaryKey=id"`
	ID       uint          `marlow:"column=id&autoIncrement=true"`
	Name     string        `marlow:"column=name"`
	ParentID sql.NullInt64 `marlow:"column=parent_id"`
}
