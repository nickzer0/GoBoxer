package handlers

import (
	"log"
	"net/http"

	"github.com/nickzer0/GoBoxer/internal/helpers"
)

// Login renders the login page.
func (m *Repository) Login(w http.ResponseWriter, r *http.Request) {
	if err := helpers.RenderPage(w, r, "login", nil, nil); err != nil {
		printTemplateError(w, err)
	}
}

// PostLogin handles the user login process, authenticating credentials.
func (m *Repository) PostLogin(w http.ResponseWriter, r *http.Request) {
	if err := m.App.Session.RenewToken(r.Context()); err != nil {
		log.Printf("Error renewing session token: %v", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error processing form")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	user, err := m.DB.Authenticate(username, password)
	if err != nil {
		log.Printf("Authentication failed for user %s: %v", username, err)
		m.App.Session.Put(r.Context(), "error", "Invalid credentials!")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Store user details in session
	m.App.Session.Put(r.Context(), "user", user)
	m.App.Session.Put(r.Context(), "username", user.Username)
	m.App.Session.Put(r.Context(), "user_id", user.ID)
	m.App.Session.Put(r.Context(), "access_level", user.AccessLevel)
	m.App.Session.Put(r.Context(), "first_name", user.FirstName)
	m.App.Session.Put(r.Context(), "last_name", user.LastName)
	m.App.Session.Put(r.Context(), "ssh_key", user.SSHKey)
	http.Redirect(w, r, "/app/home", http.StatusSeeOther)
}

// Logout handles user logout by destroying the current session and renewing the session token.
func (m *Repository) Logout(w http.ResponseWriter, r *http.Request) {
	if err := m.App.Session.Destroy(r.Context()); err != nil {
		log.Printf("Error destroying session: %v", err)
	}

	if err := m.App.Session.RenewToken(r.Context()); err != nil {
		log.Printf("Error renewing token: %v", err)
	}

	m.App.Session.Put(r.Context(), "flash", "You have been logged out successfully.")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// NotFound renders the 404 Not Found page.
func (m *Repository) NotFound(w http.ResponseWriter, r *http.Request) {
	if err := helpers.RenderPage(w, r, "404", nil, nil); err != nil {
		log.Printf("Error rendering 404 page: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}
}
