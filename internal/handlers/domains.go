package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/nickzer0/GoBoxer/internal/domains"
	"github.com/nickzer0/GoBoxer/internal/helpers"
	"github.com/nickzer0/GoBoxer/internal/models"
)

// Domains displays the domains page with a list of all domains.
func (m *Repository) Domains(w http.ResponseWriter, r *http.Request) {
	domains, err := m.DB.GetAllDomains()
	if err != nil {
		log.Printf("Error fetching domains: %v", err)
		printTemplateError(w, err)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("domains", domains)

	if err := helpers.RenderPage(w, r, "domains", vars, nil); err != nil {
		log.Printf("Error rendering domains page: %v", err)
		printTemplateError(w, err)
	}
}

// DomainsAdd displays the page for adding a new domain, including available providers.
func (m *Repository) DomainsAdd(w http.ResponseWriter, r *http.Request) {
	secrets, err := m.DB.GetAllSecrets()
	if err != nil {
		log.Printf("Error fetching secrets: %v", err)
		printTemplateError(w, err)
		return
	}

	var providers []string
	if _, ok := secrets["namecheapuser"]; ok {
		providers = append(providers, "namecheap")
	}
	if _, ok := secrets["godaddykey"]; ok {
		providers = append(providers, "godaddy")
	}

	vars := make(jet.VarMap)
	vars.Set("providers", providers)

	if err := helpers.RenderPage(w, r, "domains-add", vars, nil); err != nil {
		log.Printf("Error rendering domain add page: %v", err)
		printTemplateError(w, err)
	}
}

// DomainsLookup performs a domain lookup based on the provider and sends status updates.
func (m *Repository) DomainsLookup(w http.ResponseWriter, r *http.Request) {
	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		printTemplateError(w, err)
		return
	}

	formDomain := r.Form.Get("domain")
	formProvider := r.Form.Get("provider")

	data := map[string]string{
		"status": "checking",
	}
	m.SendStatus(userID, "domain-purchase", data)

	var domain models.Domains
	var err error

	switch formProvider {
	case "godaddy":
		domain, err = domains.Repo.GoDaddyLookupDomain(formDomain)
	case "namecheap":
		domain, err = domains.Repo.NamecheapLookupDomain(formDomain)
	default:
		err = fmt.Errorf("unsupported provider")
	}

	if err != nil {
		log.Printf("Error looking up domain with %s: %v", formProvider, err)
		m.App.Session.Put(r.Context(), "error", "Domain lookup failed")
		http.Redirect(w, r, "/domains/add", http.StatusSeeOther)
		return
	}

	data = map[string]string{
		"domain":   domain.Name,
		"provider": domain.Provider,
		"status":   domain.Status,
		"price":    domain.Price,
	}
	m.SendStatus(userID, "domain-lookup", data)

	w.Header().Set("Content-Type", "application/text")
	w.Write([]byte("done"))
}

// DomainsPurchase handles the purchase of a domain and updates the database accordingly.
func (m *Repository) DomainsPurchase(w http.ResponseWriter, r *http.Request) {
	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))
	userName := m.App.Session.Get(r.Context(), "username").(string)

	if err := r.ParseMultipartForm(0); err != nil {
		log.Printf("Error parsing form: %v", err)
		printTemplateError(w, err)
		return
	}

	formDomain := r.PostForm.Get("domain")
	formProvider := r.PostForm.Get("provider")

	// Initial status update to "checking"
	m.SendStatus(userID, "domain-purchase", map[string]string{"status": "checking"})

	// Attempt to retrieve the domain to check if it's already purchased
	_, err := m.DB.GetDomainFromDatabase(formDomain)
	if err == nil {
		log.Printf("Domain %s already exists in the database.", formDomain)
		w.Write([]byte("Domain already purchased"))
		return
	}

	// If domain not found, proceed with purchase
	if err != nil && err.Error() == "not found" {
		// Assuming GoDaddy as the provider for simplicity; extend as needed
		if formProvider == "godaddy" {
			err = domains.Repo.GoDaddyPurchaseDomain(formDomain)
			if err != nil {
				log.Printf("Error purchasing domain %s: %v", formDomain, err)
				m.SendStatus(userID, "domain-purchase", map[string]string{"status": "error"})
				return
			}

			domain := models.Domains{
				Provider:  formProvider,
				Name:      formDomain,
				CreatedBy: userName,
				Status:    "Owned",
			}
			if err = m.DB.AddDomain(domain); err != nil {
				log.Printf("Error adding domain %s to database: %v", formDomain, err)
				return
			}

			// Optionally create AWS Hosted zone for the domain
			// go domains.Repo.CreateAWSHostedZoneForDomain(domain)
		}
	}

	m.SendStatus(userID, "domain-purchase", map[string]string{"status": "complete"})
	w.Header().Set("Content-Type", "application/text")
	w.Write([]byte("done"))
}

// DomainsView displays details for a specific domain, including DNS records.
func (m *Repository) DomainsView(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 4 {
		log.Println("Invalid URL structure for domain view")
		http.Redirect(w, r, "/domains", http.StatusSeeOther)
		return
	}

	domainID, err := strconv.Atoi(exploded[3])
	if err != nil {
		log.Printf("Error converting domain ID: %v", err)
		printErrorPage(w, err)
		return
	}

	domain, err := m.DB.GetDomainById(domainID)
	if err != nil {
		log.Printf("Error fetching domain by ID: %v", err)
		printErrorPage(w, err)
		return
	}

	dnsRecords, err := domains.Repo.LookupDNSRecords(domain.Name)
	if err != nil {
		log.Printf("Error looking up DNS records for domain: %v", err)
		printErrorPage(w, err)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("domain", domain)
	vars.Set("dns", dnsRecords)

	if err := helpers.RenderPage(w, r, "domains-view", vars, nil); err != nil {
		log.Printf("Error rendering domains-view page: %v", err)
		printTemplateError(w, err)
	}
}

// DomainsEdit displays the edit page for a specific domain, including its DNS records.
func (m *Repository) DomainsEdit(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 4 {
		log.Println("Invalid URL structure for domains edit")
		http.Redirect(w, r, "/domains", http.StatusSeeOther)
		return
	}

	domainID, err := strconv.Atoi(exploded[3])
	if err != nil {
		log.Printf("Error converting domain ID: %v", err)
		printErrorPage(w, err)
		return
	}

	domain, err := m.DB.GetDomainById(domainID)
	if err != nil {
		log.Printf("Error fetching domain by ID: %v", err)
		printErrorPage(w, err)
		return
	}

	records, err := m.DB.GetDnsRecordsForDomain(domain.Name)
	if err != nil {
		log.Printf("Error fetching DNS records for domain: %v", err)
		printErrorPage(w, err)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("domain", domain)
	vars.Set("dns", records)

	if err := helpers.RenderPage(w, r, "domains-edit", vars, nil); err != nil {
		log.Printf("Error rendering domains-edit page: %v", err)
		printTemplateError(w, err)
	}
}

// TODO: This is super janky, need to rethink how we do this
// DomainsEditPost handles the POST request to update DNS records for a domain.
func (m *Repository) DomainsEditPost(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 4 {
		log.Println("Invalid URL structure for domain edit post")
		http.Redirect(w, r, "/domains", http.StatusSeeOther)
		return
	}

	domainID, err := strconv.Atoi(exploded[3])
	if err != nil {
		log.Printf("Error converting domain ID: %v", err)
		printErrorPage(w, err)
		return
	}

	domain, err := m.DB.GetDomainById(domainID)
	if err != nil {
		log.Printf("Error fetching domain by ID: %v", err)
		printErrorPage(w, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		printTemplateError(w, err)
		return
	}

	// Initialize DNS records slice for updates
	var dnsRecords []models.DNS

	// Estimate form length for iteration
	formLength := len(r.Form) / 4 // Adjusted division factor to account for each record's fields

	for i := 0; i < formLength; i++ {
		formData := r.FormValue("row" + strconv.Itoa(i) + "-data")
		formName := r.FormValue("row" + strconv.Itoa(i) + "-name")
		formType := r.FormValue("row" + strconv.Itoa(i) + "-type")
		formTtl := r.FormValue("row" + strconv.Itoa(i) + "-ttl")

		if formData == "" || formType == "" {
			continue // Skip empty or incomplete records
		}

		ttlInt, err := strconv.Atoi(formTtl)
		if err != nil {
			log.Printf("Error converting TTL for record %d: %v", i, err)
			continue // Log and skip records with invalid TTL
		}

		formName = normalizeDNSRecordName(formName, domain.Name)
		dnsRecord := models.DNS{
			Domain: domain.Name,
			Data:   formData,
			Name:   formName,
			Type:   formType,
			Ttl:    ttlInt,
		}
		dnsRecords = append(dnsRecords, dnsRecord)
	}

	// Process DNS records updates
	for _, record := range dnsRecords {
		if err := domains.Repo.AddDNSRecordToAWS(domain, record); err != nil {
			log.Printf("Error updating DNS record to AWS: %v", err)

		}
	}

	// Redirect or render confirmation
	m.App.Session.Put(r.Context(), "flash", "DNS records updated successfully.")
	http.Redirect(w, r, fmt.Sprintf("/domains/edit/%d", domainID), http.StatusSeeOther)
}

// normalizeDNSRecordName ensures the DNS record name is correctly formatted.
func normalizeDNSRecordName(name, domainName string) string {
	if name == "@" {
		return domainName
	}
	return name
}

func (m *Repository) DomainsDnsRefresh(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	domainID, err := strconv.Atoi(exploded[3])
	if err != nil {
		log.Printf("Error converting domain ID: %v", err)
		http.Error(w, "Invalid domain ID", http.StatusBadRequest)
		return
	}

	domain, err := m.DB.GetDomainById(domainID)
	if err != nil {
		log.Printf("Error fetching domain by ID: %v", err)
		http.Error(w, "Domain not found", http.StatusNotFound)
		return
	}

	domains.Repo.CreateAWSHostedZoneForDomain(domain)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("DNS refresh initiated successfully"))
}
