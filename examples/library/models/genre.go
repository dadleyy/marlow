package models

import "fmt"
import "database/sql"

// Genre records are used to group and describe a types of books.
type Genre struct {
	table    bool          `marlow:"tableName=genres&dialect=postgres&primaryKey=id"`
	ID       uint          `marlow:"column=id&autoIncrement=true"`
	Name     string        `marlow:"column=name"`
	ParentID sql.NullInt64 `marlow:"column=parent_id"`
}

func (g *Genre) String() string {
	return fmt.Sprintf("%s", g.Name)
}
