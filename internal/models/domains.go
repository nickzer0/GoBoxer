package models

import "time"

type Domains struct {
	ID         int
	Provider   string
	ProviderID string
	Name       string
	CreatedBy  string
	CreatedAt  time.Time
	Price      string
	Status     string
}
