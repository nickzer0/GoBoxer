package domains

import (
	"log"
	"strconv"

	"github.com/billputer/go-namecheap"
	"github.com/nickzer0/GoBoxer/internal/models"
)

func (m *Repository) NamecheapListDomains() ([]models.Domains, error) {
	var domains []models.Domains
	var tempDomain models.Domains

	apiName, err := m.DB.GetSecret("namecheapuser")
	if err != nil {
		return domains, err
	}
	apiKey, err := m.DB.GetSecret("namecheapkey")
	if err != nil {
		return domains, err
	}

	client := namecheap.NewClient(apiName, apiKey, apiName)
	client.BaseURL = "https://api.sandbox.namecheap.com/xml.response"

	returnedDomains, err := client.DomainsGetList()
	if err != nil {
		log.Println(err)
		return domains, err
	}

	for _, domain := range returnedDomains {
		tempDomain.ProviderID = strconv.Itoa(domain.ID)
		tempDomain.Name = domain.Name
		tempDomain.Provider = "namecheap"
		domains = append(domains, tempDomain)
	}

	return domains, nil
}

func (m *Repository) NamecheapLookupDomain(domainName string) (models.Domains, error) {
	var domain models.Domains

	apiName, err := m.DB.GetSecret("namecheapuser")
	if err != nil {
		return domain, err
	}
	apiKey, err := m.DB.GetSecret("namecheapkey")
	if err != nil {
		return domain, err
	}

	client := namecheap.NewClient(apiName, apiKey, apiName)

	info, err := client.DomainsCheck(domainName)
	if err != nil {
		log.Println(err)
	}
	log.Println(info)

	return domain, nil
}
