package dbrepo

import (
	"database/sql"
	"errors"

	"github.com/nickzer0/GoBoxer/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// Authenticate checks if a user's username and password are correct.
func (m *sqliteDBRepo) Authenticate(username, testPass string) (models.User, error) {
	var user models.User

	query := `SELECT id, username, first_name, last_name, password, access_level, ssh_key 
              FROM users 
              WHERE username = ?`
	err := m.DB.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.HashedPassword, &user.AccessLevel, &user.SSHKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return user, errors.New("username not found")
		}
		return user, err
	}

	// Compare hashed password and the test password
	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(testPass))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return user, errors.New("incorrect password")
	} else if err != nil {
		return user, err
	}

	return user, nil
}

// GetUsers returns a slice of strings of all users in DB
func (m *sqliteDBRepo) GetUsers() ([]models.User, error) {
	var users []models.User
	stmt, err := m.DB.Prepare("select id, username, first_name, last_name, password, access_level, ssh_key from users")
	if err != nil {
		return users, err
	}

	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		return users, err

	}
	defer rows.Close()
	var tempUser models.User
	for rows.Next() {
		rows.Scan(
			&tempUser.ID,
			&tempUser.Username,
			&tempUser.FirstName,
			&tempUser.LastName,
			&tempUser.HashedPassword,
			&tempUser.AccessLevel,
			&tempUser.SSHKey,
		)
		projects, err := m.GetProjectsForUser(tempUser.Username)
		if err != nil {
			return users, err
		}
		tempUser.Projects = projects
		users = append(users, tempUser)
		if err != nil {
			return users, err
		}
	}

	return users, nil

}

// GetUserIDFromUsername returns user ID from username
func (m *sqliteDBRepo) GetUserIDFromUsername(username string) (int, error) {
	var id int
	stmt, err := m.DB.Prepare("SELECT id FROM users WHERE username = ?")
	if err != nil {
		return id, err
	}

	err = m.DB.Ping()
	if err != nil {
		return id, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(username)
	if err != nil {
		return id, err
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&id)
		if err != nil {
			return id, err
		}
	}

	return id, err
}

// GetUserNameFromID returns username from ID
func (m *sqliteDBRepo) GetUserFromID(id int) (models.User, error) {
	var user models.User

	stmt, err := m.DB.Prepare("SELECT id, username, first_name, last_name, password, access_level, ssh_key FROM users WHERE id = ?")
	if err != nil {
		return user, err
	}

	defer stmt.Close()
	rows, err := stmt.Query(id)
	if err != nil {
		return user, err
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(
			&user.ID,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.HashedPassword,
			&user.AccessLevel,
			&user.SSHKey,
		)
		if err != nil {
			return user, err
		}
	}

	projects, err := m.GetProjectsForUser(user.Username)
	if err != nil {
		return user, err
	}
	user.Projects = projects

	return user, nil
}

// GetUserNameFromID returns username from ID
func (m *sqliteDBRepo) UpdateUser(user models.User) error {
	update := `
	UPDATE users 
	SET first_name = ?, last_name = ?, password = ?, access_level = ?, ssh_key = ?
	WHERE id = ?
	`

	stmt, err := m.DB.Prepare(update)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(user.FirstName, user.LastName, user.HashedPassword, user.AccessLevel, user.SSHKey, user.ID)
	if err != nil {
		return err
	}

	return nil
}

// DeleteUser deletes a user from the database
func (m *sqliteDBRepo) DeleteUser(userID int) error {
	_, err := m.DB.Exec("DELETE FROM users WHERE id = ?", userID)
	return err
}

// AddUser adds a new user to the database.
func (m *sqliteDBRepo) AddUser(user models.User) error {
	if user.SSHKey == "" {
		user.SSHKey = "0" // Assuming "0" is a placeholder value for users without an SSH key.
	}

	_, err := m.DB.Exec(`INSERT INTO users (username, first_name, last_name, password, access_level, ssh_key) VALUES (?, ?, ?, ?, ?, ?)`,
		user.Username, user.FirstName, user.LastName, user.HashedPassword, user.AccessLevel, user.SSHKey)

	return err
}
