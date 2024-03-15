package main

import (
	"encoding/gob"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nickzer0/GoBoxer/internal/config"
	"github.com/nickzer0/GoBoxer/internal/models"
)

const goBoxerVersion = "0.0.1"

var app config.AppConfig
var preferenceMap map[string]string

func init() {
	log.Printf("**********************")
	log.Printf("**\tGoBoxer v%s\t**\n", goBoxerVersion)
	log.Printf("**********************")
	gob.Register(models.User{})

}

func main() {
	port, err := setup()
	if err != nil {
		log.Fatal(err)
	}

	defer app.DB.SQL.Close()

	// create http server
	srv := &http.Server{
		Addr:              "0.0.0.0:" + port,
		Handler:           routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	// start the server
	log.Println("Webserver started on", srv.Addr)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

}
