package networking

import (
	"fmt"
	"sync"
)

// DNSResolver provides local DNS resolution for services.
type DNSResolver struct {
	records map[string]string // hostname -> IP
	mu      sync.RWMutex
}

// NewDNSResolver creates a new DNS resolver.
func NewDNSResolver() *DNSResolver {
	return &DNSResolver{
		records: make(map[string]string),
	}
}

// AddRecord adds a DNS record.
func (d *DNSResolver) AddRecord(hostname, ip string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.records[hostname] = ip
}

// RemoveRecord removes a DNS record.
func (d *DNSResolver) RemoveRecord(hostname string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.records, hostname)
}

// Resolve returns the IP for a hostname.
func (d *DNSResolver) Resolve(hostname string) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ip, ok := d.records[hostname]
	return ip, ok
}

// RegisterService registers a service with the DNS resolver.
func (d *DNSResolver) RegisterService(serviceName, containerIP, networkDomain string) {
	// Register short name: service.local
	shortName := fmt.Sprintf("%s.local", serviceName)
	d.AddRecord(shortName, containerIP)

	// Register FQDN: service.project.local
	fqdn := fmt.Sprintf("%s.%s", serviceName, networkDomain)
	d.AddRecord(fqdn, containerIP)
}

// UnregisterService removes a service from DNS.
func (d *DNSResolver) UnregisterService(serviceName, networkDomain string) {
	shortName := fmt.Sprintf("%s.local", serviceName)
	fqdn := fmt.Sprintf("%s.%s", serviceName, networkDomain)
	d.RemoveRecord(shortName)
	d.RemoveRecord(fqdn)
}

// ListRecords returns all DNS records.
func (d *DNSResolver) ListRecords() map[string]string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range d.records {
		result[k] = v
	}
	return result
}
