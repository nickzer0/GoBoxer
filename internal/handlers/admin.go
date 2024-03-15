package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/nickzer0/GoBoxer/internal/helpers"
	"github.com/nickzer0/GoBoxer/internal/models"
	"github.com/nickzer0/GoBoxer/internal/server"
	"golang.org/x/crypto/bcrypt"
)

// Settings handles the request for the settings page.
func (m *Repository) Settings(w http.ResponseWriter, r *http.Request) {
	providerKeys, err := m.DB.GetAllSecrets()
	if err != nil {
		log.Printf("Failed to get all secrets: %v", err)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("provider_keys", providerKeys)

	err = helpers.RenderPage(w, r, "settings", vars, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// SettingsEdit is the POST handler for editing settings on the settings page.
func (m *Repository) SettingsEdit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		printErrorPage(w, err)
		return // Ensure no further execution on error
	}

	// Retrieve form values
	digitalocean := r.Form.Get("digital_ocean")
	linode := r.Form.Get("linode")
	godaddykey := r.Form.Get("godaddy_key")
	godaddysecret := r.Form.Get("godaddy_secret")
	namecheapuser := r.Form.Get("namecheap_user")
	namecheapkey := r.Form.Get("namecheap_key")
	awsaccount := r.Form.Get("aws_account")
	awssecret := r.Form.Get("aws_secret")
	sshkey := r.Form.Get("ssh_key")
	sshfingerprint := r.Form.Get("ssh_fingerprint")

	// Update API keys and secrets
	m.ChangeAPIKey("digitalocean", digitalocean)
	m.ChangeAPIKey("linode", linode)
	m.ChangeAPIKey("godaddykey", godaddykey)
	m.ChangeAPIKey("godaddysecret", godaddysecret)
	m.ChangeAPIKey("namecheapuser", namecheapuser)
	m.ChangeAPIKey("namecheapkey", namecheapkey)
	m.ChangeAPIKey("awsaccount", awsaccount)
	m.ChangeAPIKey("awssecret", awssecret)
	m.ChangeAPIKey("sshkey", sshkey)
	m.ChangeAPIKey("sshfingerprint", sshfingerprint)

	m.App.Session.Put(r.Context(), "flash", "Settings saved!")
	http.Redirect(w, r, "/app/admin/settings", http.StatusSeeOther)
}

// UpdateSSH adds the root SSH key to the configured cloud providers.
func (m *Repository) UpdateSSH(w http.ResponseWriter, r *http.Request) {
	providerKeys, err := m.DB.GetAllSecrets()
	if err != nil {
		log.Printf("Error getting secrets: %v", err)
		return
	}

	// Attempt to update SSH key on Linode if a Linode token exists
	if providerKeys["linode"] != "" {
		// Placeholder for Linode SSH key update logic
		// err = updateLinodeSSHKey(providerKeys["linode"])
		// if err != nil {
		//     log.Printf("Error adding SSH key to Linode: %v", err)
		// }
	} else if providerKeys["digitalocean"] != "" {
		err = server.Repo.AddSSHKeyToDigitalOcean()
		if err != nil {
			log.Printf("Error adding SSH key to Digital Ocean: %v", err)
		}
	}
}

// ChangeAPIKey updates the value of a specified API key in the database.
func (m *Repository) ChangeAPIKey(key, value string) {
	if err := m.DB.UpdateSecret(key, value); err != nil {
		log.Printf("Failed to update secret for %s: %v", key, err)
	}
}

// Users handles the users page.
func (m *Repository) Users(w http.ResponseWriter, r *http.Request) {
	vars := make(jet.VarMap)

	users, err := m.DB.GetUsers()
	if err != nil {
		log.Printf("Error getting users: %v", err)
		printErrorPage(w, err)
		return
	}

	vars.Set("users", users)

	if err := helpers.RenderPage(w, r, "users", vars, nil); err != nil {
		printTemplateError(w, err)
	}
}

// ViewUser handles the user-view page display.
func (m *Repository) ViewUser(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	id, err := strconv.Atoi(exploded[4])
	if err != nil {
		log.Printf("Error converting ID: %v", err)
		printErrorPage(w, err)
		return
	}
	user, err := m.DB.GetUserFromID(id)
	if err != nil {
		log.Printf("Error fetching user: %v", err)
		printErrorPage(w, err)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("user", user)

	if err := helpers.RenderPage(w, r, "user-view", vars, nil); err != nil {
		printTemplateError(w, err)
	}
}

// EditUser processes the edit form for a user and updates their details in the database.
func (m *Repository) EditUser(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	id, err := strconv.Atoi(exploded[4])
	if err != nil {
		log.Printf("Error converting ID: %v", err)
		printErrorPage(w, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		printErrorPage(w, err)
		return
	}

	user, err := m.DB.GetUserFromID(id)
	if err != nil {
		log.Printf("Error fetching user: %v", err)
		printErrorPage(w, err)
		return
	}

	formFirstName := r.Form.Get("first_name")
	formLastName := r.Form.Get("last_name")
	formPassword := r.Form.Get("password")
	formAccessLevel := r.Form.Get("access_level")
	formSSHKey := r.Form.Get("ssh_key")

	// Track if any user field is updated
	updated := false

	if formFirstName != "" && formFirstName != user.FirstName {
		user.FirstName = formFirstName
		updated = true
	}

	if formLastName != "" && formLastName != user.LastName {
		user.LastName = formLastName
		updated = true
	}

	if formPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(formPassword), 12)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			printErrorPage(w, err)
			return
		}
		user.HashedPassword = string(hashedPassword)
		updated = true
	}

	if formAccessLevel != "" {
		accessLevelInt, err := strconv.Atoi(formAccessLevel)
		if err != nil {
			log.Printf("Error converting access level: %v", err)
			printErrorPage(w, err)
			return
		}
		if accessLevelInt != user.AccessLevel {
			user.AccessLevel = accessLevelInt
			updated = true
		}
	}

	if formSSHKey != "" && formSSHKey != user.SSHKey {
		user.SSHKey = formSSHKey
		updated = true
	}

	if updated {
		if err := m.DB.UpdateUser(user); err != nil {
			log.Printf("Error updating user: %v", err)
			printErrorPage(w, err)
			return
		}
		m.App.Session.Put(r.Context(), "flash", "User updated!")
	}

	http.Redirect(w, r, "/app/admin/users", http.StatusSeeOther)
}

// AddUser renders the form for adding a new user.
func (m *Repository) AddUser(w http.ResponseWriter, r *http.Request) {
	if err := helpers.RenderPage(w, r, "user-add", nil, nil); err != nil {
		printTemplateError(w, err)
	}
}

// AddUserPost handles the POST request for adding a new user.
func (m *Repository) AddUserPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		printErrorPage(w, err)
		return
	}

	password := r.Form.Get("password")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		printErrorPage(w, err)
		return
	}

	accessLevel, err := strconv.Atoi(r.Form.Get("access_level"))
	if err != nil {
		log.Printf("Error converting access level: %v", err)
		printErrorPage(w, err)
		return
	}

	user := models.User{
		Username:       r.Form.Get("username"),
		FirstName:      r.Form.Get("first_name"),
		LastName:       r.Form.Get("last_name"),
		HashedPassword: string(hashedPassword),
		AccessLevel:    accessLevel,
	}

	if err := m.DB.AddUser(user); err != nil {
		log.Printf("Error adding user: %v", err)
		printErrorPage(w, err)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "User created!")
	http.Redirect(w, r, "/app/admin/users", http.StatusSeeOther)
}

// DeleteUser handles the deletion of a user by ID.
func (m *Repository) DeleteUser(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 6 {
		log.Println("Invalid URL, user ID missing")
		printErrorPage(w, errors.New("invalid URL, user ID missing"))
		return
	}

	id, err := strconv.Atoi(exploded[5])
	if err != nil {
		log.Printf("Error converting ID: %v", err)
		printErrorPage(w, err)
		return
	}

	if err := m.DB.DeleteUser(id); err != nil {
		log.Printf("Error deleting user: %v", err)
		printErrorPage(w, err)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "User deleted!")
	http.Redirect(w, r, "/app/admin/users", http.StatusSeeOther)
}
