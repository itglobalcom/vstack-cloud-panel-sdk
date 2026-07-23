package entities

import "fmt"

// RecordType represents DNS record type
type RecordType string

const (
	RecordTypeA     RecordType = "A"
	RecordTypeAAAA  RecordType = "AAAA"
	RecordTypeMX    RecordType = "MX"
	RecordTypeCNAME RecordType = "CNAME"
	RecordTypeNS    RecordType = "NS"
	RecordTypeTXT   RecordType = "TXT"
	RecordTypeSRV   RecordType = "SRV"
)

// Protocol represents SRV record protocol
type Protocol string

const (
	SRVProtocolTCP Protocol = "TCP"
	SRVProtocolUDP Protocol = "UDP"
	SRVProtocolTLS Protocol = "TLS"
)

// TTL represents DNS record time-to-live
type TTL string

const (
	TTL1Second   TTL = "1s"
	TTL5Seconds  TTL = "5s"
	TTL30Seconds TTL = "30s"
	TTL1Minute   TTL = "1m"
	TTL5Minutes  TTL = "5m"
	TTL10Minutes TTL = "10m"
	TTL15Minutes TTL = "15m"
	TTL30Minutes TTL = "30m"
	TTL1Hour     TTL = "1h"
	TTL2Hours    TTL = "2h"
	TTL6Hours    TTL = "6h"
	TTL12Hours   TTL = "12h"
	TTL1Day      TTL = "1d"
)

// Domain represents a DNS domain
type Domain struct {
	Name        string      `json:"name"`
	IsDelegated bool        `json:"is_delegated"`
	Records     []DNSRecord `json:"records,omitempty"`
}

// DNSRecord represents a DNS record
type DNSRecord struct {
	ID   int        `json:"id,omitempty"`
	Name string     `json:"name"`
	Type RecordType `json:"type"`
	TTL  TTL        `json:"ttl"`

	// A and AAAA record fields
	IP string `json:"ip,omitempty"`

	// MX record fields
	MailHost string `json:"mail_host,omitempty"`
	Priority *int   `json:"priority,omitempty"`

	// CNAME record fields
	CanonicalName string `json:"canonical_name,omitempty"`

	// NS record fields
	NameServerHost string `json:"name_server_host,omitempty"`

	// TXT record fields
	Text string `json:"text,omitempty"`

	// SRV record fields
	Protocol *Protocol `json:"protocol,omitempty"`
	Service  string    `json:"service,omitempty"`
	Weight   *int      `json:"weight,omitempty"`
	Port     *int      `json:"port,omitempty"`
	Target   string    `json:"target,omitempty"`
}

// CreateDomainRequest represents a request to create a domain
type CreateDomainRequest struct {
	Name string `json:"name"`
}

// Validate validates the create domain request
func (r *CreateDomainRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("domain name is required")
	}
	return nil
}

// CreateRecordRequest represents a request to create a DNS record
type CreateRecordRequest struct {
	Name           string     `json:"name"`
	Type           RecordType `json:"type,omitempty"`
	TTL            TTL        `json:"ttl,omitempty"`
	IP             string     `json:"ip,omitempty"`
	MailHost       string     `json:"mail_host,omitempty"`
	Priority       *int       `json:"priority,omitempty"`
	CanonicalName  string     `json:"canonical_name,omitempty"`
	NameServerHost string     `json:"name_server_host,omitempty"`
	Text           string     `json:"text,omitempty"`
	Protocol       *Protocol  `json:"protocol,omitempty"`
	Service        string     `json:"service,omitempty"`
	Weight         *int       `json:"weight,omitempty"`
	Port           *int       `json:"port,omitempty"`
	Target         string     `json:"target,omitempty"`
}

// Validate validates the create record request
func (r *CreateRecordRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("record name is required")
	}

	// Validate based on record type
	switch r.Type {
	case RecordTypeA, RecordTypeAAAA:
		if r.IP == "" {
			return fmt.Errorf("IP address is required for %s records", r.Type)
		}
	case RecordTypeMX:
		if r.MailHost == "" {
			return fmt.Errorf("mail host is required for MX records")
		}
		if r.Priority == nil {
			return fmt.Errorf("priority is required for MX records")
		}
		if *r.Priority < 0 || *r.Priority > 65535 {
			return fmt.Errorf("priority must be between 0 and 65535")
		}
	case RecordTypeCNAME:
		if r.CanonicalName == "" {
			return fmt.Errorf("canonical name is required for CNAME records")
		}
	case RecordTypeNS:
		if r.NameServerHost == "" {
			return fmt.Errorf("name server host is required for NS records")
		}
	case RecordTypeTXT:
		if r.Text == "" {
			return fmt.Errorf("text is required for TXT records")
		}
	case RecordTypeSRV:
		if r.Service == "" {
			return fmt.Errorf("service is required for SRV records")
		}
		if r.Protocol == nil {
			return fmt.Errorf("protocol is required for SRV records")
		}
		if r.Priority == nil {
			return fmt.Errorf("priority is required for SRV records")
		}
		if *r.Priority < 0 || *r.Priority > 65535 {
			return fmt.Errorf("priority must be between 0 and 65535")
		}
		if r.Weight == nil {
			return fmt.Errorf("weight is required for SRV records")
		}
		if r.Port == nil {
			return fmt.Errorf("port is required for SRV records")
		}
		if r.Target == "" {
			return fmt.Errorf("target is required for SRV records")
		}
	default:
		if r.Type != "" {
			return fmt.Errorf("unsupported record type: %s", r.Type)
		}
	}

	return nil
}

// UpdateRecordRequest represents a request to update a DNS record
type UpdateRecordRequest struct {
	Name           string     `json:"name,omitempty"`
	Type           RecordType `json:"type,omitempty"`
	TTL            TTL        `json:"ttl,omitempty"`
	IP             string     `json:"ip,omitempty"`
	MailHost       string     `json:"mail_host,omitempty"`
	Priority       *int       `json:"priority,omitempty"`
	CanonicalName  string     `json:"canonical_name,omitempty"`
	NameServerHost string     `json:"name_server_host,omitempty"`
	Text           string     `json:"text,omitempty"`
	Protocol       *Protocol  `json:"protocol,omitempty"`
	Service        string     `json:"service,omitempty"`
	Weight         *int       `json:"weight,omitempty"`
	Port           *int       `json:"port,omitempty"`
	Target         string     `json:"target,omitempty"`
}

// Validate validates the update record request
func (r *UpdateRecordRequest) Validate() error {
	// Similar validation as CreateRecordRequest but fields are optional
	if r.Priority != nil && (*r.Priority < 0 || *r.Priority > 65535) {
		return fmt.Errorf("priority must be between 0 and 65535")
	}
	return nil
}
