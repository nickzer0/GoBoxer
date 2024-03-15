package dbrepo

import (
	"log"
	"time"

	"github.com/nickzer0/GoBoxer/internal/models"
)

// AddDomainRedirector inserts a new domain redirector into the database.
func (m *sqliteDBRepo) AddDomainRedirector(redirector models.Redirector) (models.Redirector, error) {
	query := `INSERT INTO redirectors (provider_id, provider, domain, url, status, project, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := m.DB.Exec(query, redirector.ProviderID, redirector.Provider, redirector.Domain, redirector.URL, redirector.Status, redirector.Project, time.Now())
	if err != nil {
		return redirector, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return redirector, err
	}
	redirector.ID = int(id)
	return redirector, nil
}

// RemoveDomainRedirector deletes a domain redirector from the database based on its URL.
func (m *sqliteDBRepo) RemoveDomainRedirector(redirector models.Redirector) error {
	_, err := m.DB.Exec("DELETE FROM redirectors WHERE url = ?", redirector.URL)
	if err != nil {
		log.Println("RemoveDomainRedirector error:", err)
		return err
	}
	return nil
}

// GetDomainRedirector retrieves a domain redirector by its ID from the database.
func (m *sqliteDBRepo) GetDomainRedirector(id int) (models.Redirector, error) {
	var redirector models.Redirector
	query := `SELECT id, provider_id, provider, domain, url, status, project, created_at FROM redirectors WHERE id = ?`

	err := m.DB.QueryRow(query, id).Scan(&redirector.ID, &redirector.ProviderID, &redirector.Provider, &redirector.Domain, &redirector.URL, &redirector.Status, &redirector.Project, &redirector.CreatedAt)
	if err != nil {
		return redirector, err
	}

	return redirector, nil
}

// UpdateDomainRedirector modifies the status of an existing domain redirector based on its ID.
func (m *sqliteDBRepo) UpdateDomainRedirector(redirector models.Redirector) error {
	_, err := m.DB.Exec("UPDATE redirectors SET status = ? WHERE id = ?", redirector.Status, redirector.ID)
	return err // Directly return the result of Exec, simplifying error handling
}

// GetAllDomainRedirectors retrieves all domain redirectors stored in the database.
func (m *sqliteDBRepo) GetAllDomainRedirectors() ([]models.Redirector, error) {
	var redirectors []models.Redirector
	query := "SELECT id, provider_id, provider, domain, url, status, project, created_at FROM redirectors"

	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var redirector models.Redirector
		if err := rows.Scan(&redirector.ID, &redirector.ProviderID, &redirector.Provider, &redirector.Domain, &redirector.URL, &redirector.Status, &redirector.Project, &redirector.CreatedAt); err != nil {
			return nil, err
		}
		redirectors = append(redirectors, redirector)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return redirectors, nil
}
