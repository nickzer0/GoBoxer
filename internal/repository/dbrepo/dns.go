package dbrepo

import (
	"fmt"
	"log"
	"time"

	"github.com/nickzer0/GoBoxer/internal/models"
)

// AddDnsRecord adds DNS entry to the database
func (m *sqliteDBRepo) AddDnsRecord(record models.DNS) error {
	query := `INSERT INTO dns (provider_id, domain, data, name, ttl, type, priority, weight, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := m.DB.Exec(query, record.ProviderID, record.Domain, record.Data, record.Name, record.Ttl, record.Type, record.Priority, record.Weight, time.Now())
	if err != nil {
		log.Printf("AddDnsRecord error for domain %s: %v", record.Domain, err)
		return err
	}
	return nil
}

func (m *sqliteDBRepo) AddOrIgnoreDnsRecord(record models.DNS) error {
	insert := `INSERT OR IGNORE INTO dns (provider_id, domain, data, name, ttl, type, priority, weight, created_at) 
               VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.DB.Exec(insert, record.ProviderID, record.Domain, record.Data, record.Name, record.Ttl, record.Type, record.Priority, record.Weight, time.Now())
	if err != nil {
		log.Printf("Error inserting DNS record: %v", err)
		return err
	}

	return nil
}

// GetDnsEntriesForDomain gets all the DNS entries for a domain from the database
func (m *sqliteDBRepo) GetDnsRecordsForDomain(domain string) ([]models.DNS, error) {
	var records []models.DNS

	query := `
		SELECT
			id, provider_id, domain, data, name, ttl, type, priority, weight, created_at
		FROM
			dns
		WHERE
			domain = ?
			AND type != "NS"
		`

	rows, err := m.DB.Query(query, domain)
	if err != nil {
		return records, err
	}
	defer rows.Close()

	for rows.Next() {
		var record models.DNS
		err = rows.Scan(
			&record.ID,
			&record.ProviderID,
			&record.Domain,
			&record.Data,
			&record.Name,
			&record.Ttl,
			&record.Type,
			&record.Priority,
			&record.Weight,
			&record.CreatedAt,
		)
		if err != nil {
			return records, err
		}
		records = append(records, record)
	}
	err = rows.Err()
	if err != nil {
		return records, err
	}

	return records, nil
}

// GetDnsEntryByID gets a single DNS entry from database based on ID
func (m *sqliteDBRepo) GetDnsRecordByID(id int) (models.DNS, error) {
	var record models.DNS
	query := `
	SELECT
		id, provider_id, domain, data, name, ttl, type, priority, weight, created_at
	FROM
		dns
	WHERE
		id = ?
	`

	rows := m.DB.QueryRow(query, id)
	err := rows.Scan(
		&record.ID,
		&record.ProviderID,
		&record.Domain,
		&record.Data,
		&record.Name,
		&record.Ttl,
		&record.Type,
		&record.Priority,
		&record.Weight,
		&record.CreatedAt,
	)

	if err != nil {
		return record, err
	}

	err = rows.Err()
	if err != nil {
		return record, err
	}
	return record, nil

}

// UpdateDnsRecord updates a record in the database
func (m *sqliteDBRepo) UpdateDnsRecord(record models.DNS) error {
	update := `
	UPDATE 
		dns
	SET
		provider_id = ?, domain = ?, data = ?, name = ?, ttl = ?, type = ?, priority = ?, weight = ?, created_at = ?
	WHERE
		id = ?
	`

	stmt, err := m.DB.Prepare(update)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(record.ProviderID, record.Domain, record.Data, record.Name, record.Ttl, record.Type, record.Priority, record.Weight, time.Now(), record.ID)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil

}

// DeleteDnsRecord deletes the DNS record from the database
func (m *sqliteDBRepo) DeleteDnsRecord(id int) error {
	stmt, err := m.DB.Prepare("DELETE FROM dns WHERE id = ?")
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}

	return nil
}

// DeleteDnsRecordsForDomain deletes all the records for a domain from the database
func (m *sqliteDBRepo) DeleteDnsRecordsForDomain(domain string) error {
	stmt, err := m.DB.Prepare("DELETE FROM dns WHERE domain = ?")
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec(domain)
	if err != nil {
		return err
	}

	return nil
}

// GetAWSHostedZone gets the hosted zone ID for a domain from the database
func (m *sqliteDBRepo) GetAWSHostedZone(domain string) (string, error) {
	var zoneID string
	query := `
	SELECT
		provider_id	
	FROM
		dns
	WHERE
		domain = ?
	AND
		type = 'NS'
	`

	rows := m.DB.QueryRow(query, domain)
	err := rows.Scan(&zoneID)

	if err != nil {
		return zoneID, err
	}

	err = rows.Err()
	if err != nil {
		return zoneID, err
	}
	return zoneID, nil
}
