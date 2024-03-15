package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/digitalocean/godo"
	"github.com/nickzer0/GoBoxer/internal/models"
	"golang.org/x/oauth2"
)



// TokenSource is an oauth2.TokenSource which returns a static access token
type TokenSource struct {
	AccessToken string
}

// Token returns the access token
func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

// DigitalOceanCreateServer creates a droplet on Digital Ocean and returns the server object with the IP address
func (m *Repository) DigitalOceanCreateServer(server models.Server) (models.Server, error) {
	region := "nyc3"
	size := "s-1vcpu-1gb"

	// Map server.OS to DigitalOcean's slug format
	var operatingSystem string
	switch server.OS {
	case "ubuntu-2204":
		operatingSystem = "ubuntu-22-04-x64"
	// Add more cases as needed
	default:
		return server, fmt.Errorf("unsupported OS: %s", server.OS)
	}

	// Retrieve API key and SSH fingerprint from secrets
	providerKeys, err := m.DB.GetAllSecrets()
	if err != nil {
		return server, fmt.Errorf("error getting secrets from database: %v", err)
	}
	apiKey, ok := providerKeys["digitalocean"]
	if !ok {
		return server, errors.New("digitalocean API key not found in secrets")
	}

	sshFingerprint, ok := providerKeys["sshfingerprint"]
	if !ok {
		return server, errors.New("SSH fingerprint not found in secrets")
	}

	// Initialize DigitalOcean client
	client := godo.NewFromToken(apiKey)

	// Prepare droplet creation request
	createRequest := &godo.DropletCreateRequest{
		Name:    server.Name,
		Tags:    Tags,
		SSHKeys: []godo.DropletCreateSSHKey{{Fingerprint: sshFingerprint}},
		Region:  region,
		Size:    size,
		Image:   godo.DropletCreateImage{Slug: operatingSystem},
	}

	// Create droplet
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	droplet, _, err := client.Droplets.Create(ctx, createRequest)
	if err != nil {
		return server, fmt.Errorf("error creating droplet on DigitalOcean: %v", err)
	}

	// Poll for IP address assignment
	for attempt := 0; attempt < 12; attempt++ {
		time.Sleep(5 * time.Second) // Wait before each attempt
		droplet, _, err = client.Droplets.Get(ctx, droplet.ID)
		if err != nil {
			return server, fmt.Errorf("error getting droplet from DigitalOcean: %v", err)
		}
		if vpsIP, err := droplet.PublicIPv4(); err == nil && vpsIP != "" {
			server.IP = vpsIP
			server.ProviderID = droplet.ID // Assume droplet.ID is int, adjust if it's not
			return server, nil
		}
	}

	return server, errors.New("failed to obtain public IP address for the droplet")
}

// DigitalOceanRefreshVPS refreshes the list of servers currently deployed on Digital Ocean
func (m *Repository) DigitalOceanRefreshVPS() ([]models.Server, error) {
	servers := []models.Server{}

	apiKey, err := m.DB.GetSecret("digitalocean")
	if err != nil {
		return servers, err
	}

	client := godo.NewFromToken(apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(ctx, opt)
		if err != nil {
			return servers, fmt.Errorf("error getting list of droplets from digital ocean: %v", err)
		}

		for _, droplet := range droplets {
			for _, tag := range droplet.Tags {
				if tag == Tags[0] {
					ip, err := droplet.PublicIPv4()
					if err != nil {
						return servers, fmt.Errorf("error getting IPv4 address from digital ocean: %v", err)
					}

					server := models.Server{
						Provider: "digitalocean",
						Name:     droplet.Name,
						ID:       droplet.ID,
						IP:       ip,
						Status:   droplet.Status,
					}
					servers = append(servers, server)
				}
			}

		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return servers, fmt.Errorf("error with pages listing digital ocean instances: %v", err)
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}
	return servers, nil
}

// DigitalOceanDeleteServer deletes a server from digital ocean based on serverID
func (m *Repository) DigitalOceanDeleteServer(serverID int) error {
	apiKey, err := m.DB.GetSecret("digitalocean")
	if err != nil {
		return fmt.Errorf("error getting secrets from database: %v", err)
	}

	if apiKey == "" {
		return nil
	}

	client := godo.NewFromToken(apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	opt := &godo.ListOptions{}
	droplets, _, err := client.Droplets.List(ctx, opt)
	if err != nil {
		return fmt.Errorf("error getting list of droplets from digital ocean: %v", err)
	}

	for _, droplet := range droplets {
		for _, tag := range droplet.Tags {
			if tag == Tags[0] {
				if droplet.ID == serverID {
					_, err := client.Droplets.Delete(ctx, droplet.ID)
					if err != nil {
						return fmt.Errorf("error deleting droplet from digital ocean: %v", err)
					}
				}
			}
		}

	}

	return nil
}

// DigitalOceanDeleteAll deletes all servers created in digital ocean with specified tag
func (m *Repository) DigitalOceanDeleteAll() error {
	providerKeys, err := m.DB.GetAllSecrets()
	if err != nil {
		return fmt.Errorf("error getting secrets from database: %v", err)
	}
	apiKey := providerKeys["digitalocean"]

	if apiKey == "" {
		return nil
	}

	client := godo.NewFromToken(apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	opt := &godo.ListOptions{}
	droplets, _, err := client.Droplets.List(ctx, opt)
	if err != nil {
		return fmt.Errorf("error getting list of droplets from digital ocean: %v", err)
	}
	for _, droplet := range droplets {
		for _, tag := range droplet.Tags {
			if tag == Tags[0] {
				_, err := client.Droplets.Delete(ctx, droplet.ID)
				if err != nil {
					return fmt.Errorf("error deleting droplet from digital ocean: %v", err)
				}
			}
		}
	}

	return nil
}

// AddSSHKey removes all SSH keys from the account and adds a new one
func (m *Repository) AddSSHKeyToDigitalOcean() error {
	providerKeys, err := m.DB.GetAllSecrets()
	if err != nil {
		return err
	}
	apiKey := providerKeys["digitalocean"]
	tokenSource := &TokenSource{
		AccessToken: apiKey,
	}

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := godo.NewClient(oauthClient)

	// List existing keys on Digital Ocean
	keys, _, err := client.Keys.List(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("error getting list of keys from DigitalOcean: %v", err)
	}

	// Delete those keys before adding new one
	for _, key := range keys {
		if key.Name == "Root Key" {
			_, err := client.Keys.DeleteByID(context.Background(), key.ID)
			if err != nil {
				return fmt.Errorf("error deleting key from DigitalOcean: %v", err)
			}
		}
	}

	// Now add new key
	createRequest := &godo.KeyCreateRequest{
		Name:      "Root Key",
		PublicKey: providerKeys["sshkey"],
	}

	key, _, err := client.Keys.Create(context.Background(), createRequest)
	if err != nil {
		return fmt.Errorf("error adding SSH key to DigitalOcean: %v", err)
	}

	err = m.DB.UpdateSecret("sshfingerprint", key.Fingerprint)
	if err != nil {
		return fmt.Errorf("error adding SSH key fingerprint to database: %v", err)
	}
	return nil
}
