package models

import "time"

type Script struct {
	ID          int
	Name        string
	Icon        string
	Description string
	CreatedBy   string
	CreatedAt   time.Time
}
