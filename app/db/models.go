// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package db

import ()

type User struct {
	ID        string
	Name      string
	LastScore int64
	TopScore  int64
	CreatedAt string
	UpdatedAt string
}
