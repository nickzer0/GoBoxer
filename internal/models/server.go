package models

import "time"

// Server is the model for VPS/servers created
// within the application
type Server struct {
	ID         int
	ProviderID int
	OS         string
	Provider   string
	Name       string
	IP         string
	Status     string
	Roles      []string
	Project    int
	Creator    string
	CreatedAt  time.Time
}
