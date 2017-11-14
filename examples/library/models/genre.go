package models

type Genre struct {
	table bool   `marlow:"tableName=genres&dialect=postgres"`
	ID    uint   `marlow:"column=id"`
	Name  string `marlow:"column=name"`
}
