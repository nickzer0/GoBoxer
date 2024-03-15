package templates

import "github.com/nickzer0/GoBoxer/internal/models"

// holds data sent from handlers to templates
type TemplateData struct {
	CSRFToken       string
	Flash           string
	Warning         string
	Error           string
	IsAuthenticated int
	IsAdmin         int
	Projects        []models.Project
	PreferenceMap   map[string]string
	User            models.User
}
