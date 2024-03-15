package models

import "time"

// DNS is a single DNS entry for a domain
type DNS struct {
	ID         int
	ProviderID string
	Domain     string
	Data       string
	Name       string
	Ttl        int
	Type       string
	Priority   int
	Weight     int
	CreatedAt  time.Time
}
