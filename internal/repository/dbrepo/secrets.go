package dbrepo

import (
	"time"
)

// GetAllSecrets retrieves all secrets from the database and returns them as a map.
func (m *sqliteDBRepo) GetAllSecrets() (map[string]string, error) {
	secrets := make(map[string]string)
	query := "SELECT name, value FROM secrets"
	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var name, value string
	for rows.Next() {
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		secrets[name] = value
	}

	return secrets, rows.Err()
}

// GetSecret fetches the value of a secret by its name.
func (m *sqliteDBRepo) GetSecret(name string) (string, error) {
	var value string
	err := m.DB.QueryRow("SELECT value FROM secrets WHERE name = ?", name).Scan(&value)
	return value, err
}

// UpdateSecret updates or inserts a new secret into the database.
func (m *sqliteDBRepo) UpdateSecret(name, value string) error {
	query := `INSERT INTO secrets (name, value, last_change) VALUES (?, ?, ?) ON CONFLICT(name) DO UPDATE SET value = excluded.value, last_change = excluded.last_change`
	_, err := m.DB.Exec(query, name, value, time.Now())
	return err
}
