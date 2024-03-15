package models

import "time"

// Redirector is the model for a domain redirector service
type Redirector struct {
	ID         int
	ProviderID string
	Provider   string
	URL        string
	Domain     string
	Status     string
	Project    int
	CreatedAt  time.Time
}
