package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/nickzer0/GoBoxer/internal/handlers"
)

func routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.Recoverer)
	// mux.Use(NoSurf) // TODO: Implement CSRF 
	mux.Use(SessionLoad)

	// Public routes
	mux.Get("/", handlers.Repo.Home)
	mux.Get("/login", handlers.Repo.Login)
	mux.Post("/login", handlers.Repo.PostLogin)
	mux.Get("/user/logout", handlers.Repo.Logout)
	mux.NotFound(handlers.Repo.NotFound)

	// WebSocket route
	mux.Get("/ws", handlers.Repo.ServeWs)

	// Authenticated routes
	mux.Route("/app", func(mux chi.Router) {
		mux.Use(Auth)
		mux.Get("/home", handlers.Repo.Home)
		mux.Get("/account", handlers.Repo.Account)
		mux.Post("/account", handlers.Repo.EditAccount)

		// Projects
		mux.Get("/projects", handlers.Repo.Projects)
		mux.Get("/projects/{id}", handlers.Repo.ViewProject)
		mux.Get("/projects/add", handlers.Repo.AddProjects)
		mux.Post("/projects/add", handlers.Repo.AddProjectsPost)
		mux.Post("/projects/update", handlers.Repo.UpdateProjectsPost)
		mux.Get("/projects/remove/{id}", handlers.Repo.RemoveProject)

		// Server routes
		mux.Get("/servers", handlers.Repo.Servers)
		mux.Get("/servers/add", handlers.Repo.ServersAdd)
		mux.Post("/servers/add", handlers.Repo.ServersAddPost)
		mux.Get("/servers/remove/{id}", handlers.Repo.ServersRemove)
		mux.Get("/servers/removeall", handlers.Repo.DeleteAllServers)
		mux.Get("/servers/provision/{id}", handlers.Repo.ProvisionServer)
		mux.Get("/servers/{id}", handlers.Repo.ViewServer)
		mux.Post("/servers/update/{id}", handlers.Repo.UpdateServer)

		// Script routes
		mux.Get("/scripts", handlers.Repo.Scripts)
		mux.Get("/scripts/{id}", handlers.Repo.ScriptView)
		mux.Get("/scripts/add", handlers.Repo.ScriptsAdd)
		mux.Post("/scripts/upload", handlers.Repo.UploadScript)
		mux.Post("/scripts/update/{id}", handlers.Repo.UpdateScript)
		mux.Get("/scripts/remove/{id}", handlers.Repo.RemoveScript)

		// Domain routes
		mux.Get("/domains", handlers.Repo.Domains)
		mux.Get("/domains/add", handlers.Repo.DomainsAdd)
		mux.Post("/domains/lookup", handlers.Repo.DomainsLookup)
		mux.Post("/domains/purchase", handlers.Repo.DomainsPurchase)
		mux.Get("/domains/{id}", handlers.Repo.DomainsView)
		mux.Get("/domains/{id}/edit", handlers.Repo.DomainsEdit)
		mux.Post("/domains/{id}/edit", handlers.Repo.DomainsEditPost)
		mux.Post("/domains/{id}/refresh-dns", handlers.Repo.DomainsDnsRefresh)

		// Redirectors routes
		mux.Get("/redirectors", handlers.Repo.Redirectors)
		mux.Get("/redirectors/add", handlers.Repo.RedirectorAdd)
		mux.Post("/redirectors/add", handlers.Repo.RedirectorAddPost)
		mux.Post("/redirectors/delete", handlers.Repo.RedirectorDelete)
		mux.Post("/redirectors/resync", handlers.Repo.RedirectorsSync)

		// Admin routes
		mux.Route("/admin", func(mux chi.Router) {
			mux.Use(Admin)
			// Settings routes
			mux.Get("/settings", handlers.Repo.Settings)
			mux.Get("/settings/update-ssh", handlers.Repo.UpdateSSH)
			mux.Post("/settings", handlers.Repo.SettingsEdit)

			// User routes
			mux.Get("/users", handlers.Repo.Users)
			mux.Get("/users/{id}", handlers.Repo.ViewUser)
			mux.Post("/users/{id}", handlers.Repo.EditUser)
			mux.Get("/users/add", handlers.Repo.AddUser)
			mux.Post("/users/add", handlers.Repo.AddUserPost)
			mux.Get("/users/delete/{id}", handlers.Repo.DeleteUser)
		})

	})

	// File server routes
	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))

	return mux
}
