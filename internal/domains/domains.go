package domains

import (
	"github.com/nickzer0/GoBoxer/internal/config"
	"github.com/nickzer0/GoBoxer/internal/driver"
	"github.com/nickzer0/GoBoxer/internal/repository"
	"github.com/nickzer0/GoBoxer/internal/repository/dbrepo"
)

var Repo *Repository
var app *config.AppConfig

type Repository struct {
	App *config.AppConfig
	DB  repository.DatabaseRepo
}

// NewHandlers sets the repository for the handlers
func NewDomains(r *Repository, a *config.AppConfig) {
	Repo = r
	app = a
}

// NewRepo sets the db repo for SQLite3
func NewRepo(a *config.AppConfig, db *driver.DB) *Repository {
	return &Repository{
		App: a,
		DB:  dbrepo.NewSqliteRepo(db.SQL, a),
	}
}
