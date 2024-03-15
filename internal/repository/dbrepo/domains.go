package dbrepo

import (
	"errors"
	"time"

	"github.com/nickzer0/GoBoxer/internal/models"
)

// GetAllDomains returns all domains from the database
func (m *sqliteDBRepo) GetAllDomains() ([]models.Domains, error) {
	var domains []models.Domains

	stmt := "SELECT id, provider, provider_id, name, created_by, created_at FROM domains"

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var domain models.Domains
		err = rows.Scan(
			&domain.ID,
			&domain.Provider,
			&domain.ProviderID,
			&domain.Name,
			&domain.CreatedBy,
			&domain.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return domains, nil
}

// AddDomain inserts a new domain record into the database.
func (m *sqliteDBRepo) AddDomain(domain models.Domains) error {
	query := `INSERT INTO domains (provider, provider_id, name, created_by, created_at, status) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := m.DB.Exec(query, domain.Provider, domain.ProviderID, domain.Name, domain.CreatedBy, time.Now(), domain.Status)
	if err != nil {
		return err
	}
	return nil
}

// GetDomainFromDatabase retrieves a domain by its name from the database.
func (m *sqliteDBRepo) GetDomainFromDatabase(name string) (models.Domains, error) {
	var domain models.Domains
	query := "SELECT id, provider, provider_id, name, created_by, created_at FROM domains WHERE name = ?"
	err := m.DB.QueryRow(query, name).Scan(&domain.ID, &domain.Provider, &domain.ProviderID, &domain.Name, &domain.CreatedBy, &domain.CreatedAt)
	if err != nil {
		return domain, errors.New("not found") // specific error is checked on return to prevent attempts to double purchase
	}
	return domain, nil
}

// GetDomainById fetches a domain from the database using its ID.
func (m *sqliteDBRepo) GetDomainById(id int) (models.Domains, error) {
	var domain models.Domains
	query := "SELECT id, provider, provider_id, name, status, created_by, created_at FROM domains WHERE id = ?"
	err := m.DB.QueryRow(query, id).Scan(&domain.ID, &domain.Provider, &domain.ProviderID, &domain.Name, &domain.Status, &domain.CreatedBy, &domain.CreatedAt)
	if err != nil {
		return domain, err
	}
	return domain, nil
}
