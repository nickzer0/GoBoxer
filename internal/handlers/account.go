package handlers

import (
	"log"
	"net/http"

	"github.com/CloudyKit/jet/v6"
	"github.com/nickzer0/GoBoxer/internal/helpers"
	"golang.org/x/crypto/bcrypt"
)

// Account renders the user's account details page.
func (m *Repository) Account(w http.ResponseWriter, r *http.Request) {
	id := m.App.Session.Get(r.Context(), "user_id").(int)
	user, err := m.DB.GetUserFromID(id)
	if err != nil {
		log.Printf("Error fetching user from ID: %v", err)
		printErrorPage(w, err)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("user", user)
	if err := helpers.RenderPage(w, r, "account", vars, nil); err != nil {
		log.Printf("Error rendering account page: %v", err)
		printTemplateError(w, err)
	}
}

// EditUser is POST handler for user-view page
func (m *Repository) EditAccount(w http.ResponseWriter, r *http.Request) {
	id := m.App.Session.Get(r.Context(), "user_id").(int)
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		printErrorPage(w, err)
	}

	user, err := m.DB.GetUserFromID(id)
	if err != nil {
		log.Println(err)
		printErrorPage(w, err)
	}
	formFirstName := r.Form.Get("first_name")
	formLastName := r.Form.Get("last_name")
	formPassword := r.Form.Get("password")
	sshKey := r.Form.Get("ssh_key")

	if formFirstName != "" {
		user.FirstName = formFirstName
	}

	if formLastName != "" {
		user.LastName = formLastName
	}

	if formPassword != "" {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(formPassword), 12)
		user.HashedPassword = string(hashedPassword)
	}

	if sshKey != user.SSHKey {
		user.SSHKey = sshKey
	}

	err = m.DB.UpdateUser(user)
	if err != nil {
		log.Println(err)
		printErrorPage(w, err)
	}

	m.App.Session.Put(r.Context(), "ssh_key", sshKey)

	m.App.Session.Put(r.Context(), "flash", "User updated!")
	http.Redirect(w, r, "/app/account", http.StatusSeeOther)
}
