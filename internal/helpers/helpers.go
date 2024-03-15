package helpers

import (
	"github.com/nickzer0/GoBoxer/internal/config"
	"github.com/nickzer0/GoBoxer/internal/repository"
)

var app *config.AppConfig

// NewHelpers sets up app config for helpers
func NewHelpers(a *config.AppConfig) {
	app = a
}

type Repository struct {
	App *config.AppConfig
	DB  repository.DatabaseRepo
}
