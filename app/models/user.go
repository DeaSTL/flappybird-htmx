package models

import "time"

type User struct {
	ID        string
	Name      string
	TopScore  int
	LastScore int
	CreatedAt time.Time
	UpdatedAt time.Time
}
