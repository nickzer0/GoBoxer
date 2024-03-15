package domains

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/nickzer0/GoBoxer/internal/models"
	"github.com/nickzer0/godaddy-domainclient"
)

func (m *Repository) GoDaddyListDomains() ([]models.Domains, error) {
	var domains []models.Domains
	var tempDomain models.Domains
	ctx := context.TODO()

	apiKey, err := m.DB.GetSecret("godaddykey")
	if err != nil {
		return domains, err
	}
	apiSecret, err := m.DB.GetSecret("godaddysecret")
	if err != nil {
		return domains, err
	}

	var apiConfig = godaddy.NewConfiguration()
	apiConfig.BasePath = "https://api.ote-godaddy.com"
	var authString = fmt.Sprintf("sso-key %s:%s", apiKey, apiSecret)
	apiConfig.AddDefaultHeader("Authorization", authString)
	var apiClient = godaddy.NewAPIClient(apiConfig)

	returnedDomains, _, err := apiClient.V1Api.List(ctx, nil)
	if err != nil {
		log.Println(err)
	}
	for _, domain := range returnedDomains {
		tempDomain.ProviderID = strconv.Itoa(int(domain.DomainId))
		tempDomain.Name = domain.Domain
		tempDomain.Provider = "godaddy"
		domains = append(domains, tempDomain)
	}

	return domains, nil
}

func (m *Repository) GoDaddyLookupDomain(domainName string) (models.Domains, error) {
	var domain models.Domains
	ctx := context.TODO()

	apiKey, err := m.DB.GetSecret("godaddykey")
	if err != nil {
		return domain, err
	}
	apiSecret, err := m.DB.GetSecret("godaddysecret")
	if err != nil {
		return domain, err
	}

	var apiConfig = godaddy.NewConfiguration()
	apiConfig.BasePath = "https://api.ote-godaddy.com"
	var authString = fmt.Sprintf("sso-key %s:%s", apiKey, apiSecret)
	apiConfig.AddDefaultHeader("Authorization", authString)
	var apiClient = godaddy.NewAPIClient(apiConfig)

	returnedDomain, _, err := apiClient.V1Api.Available(ctx, domainName, nil)
	if err != nil {
		log.Println(err)
	}

	domain.Name = returnedDomain.Domain
	if returnedDomain.Available {
		domain.Status = "Available"
	} else {
		domain.Status = "Taken"
	}

	price := float64(returnedDomain.Price)
	convertedPrice := fmt.Sprintf(("%.2f"), price/10000000)
	domain.Price = convertedPrice

	domain.Provider = "GoDaddy"

	return domain, nil
}

func (m *Repository) GoDaddyPurchaseDomain(domainName string) error {
	ctx := context.TODO()

	apiKey, err := m.DB.GetSecret("godaddykey")
	if err != nil {
		return err
	}
	apiSecret, err := m.DB.GetSecret("godaddysecret")
	if err != nil {
		return err
	}

	var apiConfig = godaddy.NewConfiguration()
	apiConfig.BasePath = "https://api.ote-godaddy.com"
	var authString = fmt.Sprintf("sso-key %s:%s", apiKey, apiSecret)
	apiConfig.AddDefaultHeader("Authorization", authString)
	var apiClient = godaddy.NewAPIClient(apiConfig)

	consent := &godaddy.Consent{
		AgreedBy:      "123.123.123.123", // Public IP address
		AgreementKeys: []string{"DNRA", "DNPA"},
	}

	address := &godaddy.Address{
		Address1:   "Test",
		City:       "Test",
		PostalCode: "E1 6AN",
		Country:    "GB",
		State:      "Testing",
	}

	contact := &godaddy.Contact{
		AddressMailing: address,
		NameFirst:      "Test",
		NameLast:       "Test",
		Email:          "test@email.com",
		Phone:          "+44.07889889889",
	}

	purchase := godaddy.DomainPurchase{
		Consent:           consent,
		ContactAdmin:      contact,
		ContactBilling:    contact,
		ContactRegistrant: contact,
		ContactTech:       contact,
		Domain:            domainName,
		Privacy:           true,
	}

	resp, _, err := apiClient.V1Api.Purchase(ctx, purchase, nil)
	if err != nil {
		log.Println("error:", err)
	}
	log.Println("got response:", resp)

	return nil

}

func (m *Repository) GoDaddyUpdateNameServer(domain string, nameServers []*string) error {
	ctx := context.TODO()

	apiKey, err := m.DB.GetSecret("godaddykey")
	if err != nil {
		return err
	}
	apiSecret, err := m.DB.GetSecret("godaddysecret")
	if err != nil {
		return err
	}

	var apiConfig = godaddy.NewConfiguration()
	apiConfig.BasePath = "https://api.ote-godaddy.com"
	var authString = fmt.Sprintf("sso-key %s:%s", apiKey, apiSecret)
	apiConfig.AddDefaultHeader("Authorization", authString)
	var apiClient = godaddy.NewAPIClient(apiConfig)

	var records []godaddy.DnsRecord

	for _, record := range nameServers {
		if !strings.Contains(*record, ".com") && !strings.Contains(*record, ".org") { // TODO: This can be used to stop sending certain NS, sometimes GoDaddy does like certain TLDs
			records = append(records, godaddy.DnsRecord{
				Data:  *record,
				Name:  "Name Server",
				Ttl:   600,
				Type_: "NS",
			})
		}

	}

	for i := 0; i < 10; i++ {
		log.Println("Updating nameservers....")
		_, err = apiClient.V1Api.RecordReplace(ctx, domain, records, nil)
		if err != nil {
			log.Println(i, "- Error! Trying again in 10 seconds...")
			time.Sleep(time.Second * 10)
		} else {
			break
		}
	}

	log.Println("Updated name servers on GoDaddy")

	return nil
}

// func (m *Repository) GoDaddyGetAllDNS(domain string) ([]models.DNSEntry, error) {
// 	ctx := context.TODO()

// 	// recordTypes := []string{"A", "CNAME", "PTR", "NS", "MX", "TXT"}

// 	apiKey, err := m.DB.GetSecretByName("godaddykey")
// 	if err != nil {
// 		return nil, err
// 	}
// 	apiSecret, err := m.DB.GetSecretByName("godaddysecret")
// 	if err != nil {
// 		return nil, err
// 	}

// 	var apiConfig = godaddy.NewConfiguration()
// 	apiConfig.BasePath = "https://api.ote-godaddy.com"
// 	var authString = fmt.Sprintf("sso-key %s:%s", apiKey, apiSecret)
// 	apiConfig.AddDefaultHeader("Authorization", authString)
// 	var apiClient = godaddy.NewAPIClient(apiConfig)

// 	records, _, err := apiClient.V1Api.RecordGetAll(ctx, domain, nil)
// 	if err != nil {
// 		log.Println(err)
// 		return nil, err
// 	}

// 	var recordsToReturn []models.DNSEntry
// 	for _, rec := range records {
// 		var record models.DNSEntry
// 		record.Domain = domain
// 		record.Data = rec.Data
// 		record.Name = rec.Name
// 		record.Ttl = rec.Ttl
// 		record.Type = rec.Type_
// 		if rec.Type_ == "MX" {
// 			record.Priority = rec.Priority
// 			record.Weight = rec.Weight
// 		}
// 		recordsToReturn = append(recordsToReturn, record)
// 	}
// 	return recordsToReturn, nil
// }
