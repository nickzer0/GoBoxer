package domains

import (
	"net"

	"github.com/nickzer0/GoBoxer/internal/models"
)

func (m *Repository) LookupDNSRecords(domain string) ([]models.DNS, error) {
	var dnsRecords []models.DNS

	iprecords, err := net.LookupIP(domain)
	if err == nil {
		for _, ip := range iprecords {
			var dnsRecord models.DNS
			dnsRecord.Type = "A"
			dnsRecord.Data = ip.String()
			dnsRecord.Name = domain
			dnsRecords = append(dnsRecords, dnsRecord)
		}
	}

	cname, err := net.LookupCNAME(domain)
	if err == nil {
		var dnsRecord models.DNS
		dnsRecord.Type = "CNAME"
		dnsRecord.Data = cname
		dnsRecord.Name = cname
		dnsRecords = append(dnsRecords, dnsRecord)
	}

	ptr, err := net.LookupAddr(domain)
	if err == nil {
		for _, ptrvalue := range ptr {
			var dnsRecord models.DNS
			dnsRecord.Type = "PTR"
			dnsRecord.Data = ptrvalue
			dnsRecords = append(dnsRecords, dnsRecord)
		}
	}

	nameserver, err := net.LookupNS(domain)
	if err == nil {
		for _, ns := range nameserver {
			var dnsRecord models.DNS
			dnsRecord.Type = "NS"
			dnsRecord.Data = ns.Host
			dnsRecords = append(dnsRecords, dnsRecord)
		}
	}

	mxrecords, err := net.LookupMX(domain)
	if err == nil {
		for _, mx := range mxrecords {
			var dnsRecord models.DNS
			dnsRecord.Type = "MX"
			dnsRecord.Data = mx.Host
			dnsRecords = append(dnsRecords, dnsRecord)
		}
	}

	txtrecords, err := net.LookupTXT(domain)
	if err == nil {
		for _, txt := range txtrecords {
			var dnsRecord models.DNS
			dnsRecord.Type = "TXT"
			dnsRecord.Data = txt
			dnsRecords = append(dnsRecords, dnsRecord)
		}
	}
	return dnsRecords, nil
}
