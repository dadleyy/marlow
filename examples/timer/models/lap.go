package models

import "time"

type Lap struct {
	DestroyedAt *time.Time `marlow:"column=destroyed_at"`
}
