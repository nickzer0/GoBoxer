package dbrepo

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nickzer0/GoBoxer/internal/config"
	"github.com/nickzer0/GoBoxer/internal/repository"
)

type sqliteDBRepo struct {
	App *config.AppConfig
	DB  *sql.DB
}

func NewSqliteRepo(conn *sql.DB, a *config.AppConfig) repository.DatabaseRepo {
	return &sqliteDBRepo{
		App: a,
		DB:  conn,
	}
}
