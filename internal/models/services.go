package models

// Services is the model to keep track of how many
// items are deployed to each service
type Services struct {
	DigitalOcean int
	Linode       int
	AWS          int
	NameCheap    int
	GoDaddy      int
	Azure        int
}
