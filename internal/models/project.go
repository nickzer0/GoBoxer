package models

import "time"

// Project is the model for projects created
// within the application
type Project struct {
	ID            int
	ProjectNumber int
	ProjectName   string
	CreatedBy     string
	Notes         string
	CreatedAt     time.Time
	AssignedTo    []string
}
