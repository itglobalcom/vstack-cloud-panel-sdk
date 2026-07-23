package sdk

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	domainsBaseURL = "domains"
	recordsPath    = "records"
)

// Response types for DNS operations
type (
	// GetDomainResponse represents a single domain response
	GetDomainResponse struct {
		Domain *entities.Domain `json:"domain,omitempty"`
	}

	// ListDomainsResponse represents a list of domains response
	ListDomainsResponse struct {
		Domains []*entities.Domain `json:"domains,omitempty"`
	}

	// GetRecordResponse represents a single record response
	GetRecordResponse struct {
		Record *entities.DNSRecord `json:"record,omitempty"`
	}

	// ListRecordsResponse represents a list of records response
	ListRecordsResponse struct {
		Records []entities.DNSRecord `json:"records,omitempty"`
	}
)

// buildDomainPath constructs the path for domain operations
func buildDomainPath(domainName string, parts ...string) string {
	if domainName == "" {
		return domainsBaseURL
	}

	path := fmt.Sprintf("%s/%s", domainsBaseURL, domainName)
	for _, part := range parts {
		if part != "" {
			path = fmt.Sprintf("%s/%s", path, part)
		}
	}
	return path
}

// normalizeDomainName converts a DNS name to the API's canonical form: lower
// case + trailing dot (the API stores names exactly this way; DNS names are
// case-insensitive). Applied to all paths and request names so that the
// value sent and the value read match.
func normalizeDomainName(domain string) string {
	if domain == "" {
		return domain
	}

	domain = strings.ToLower(domain)
	if !strings.HasSuffix(domain, ".") {
		return domain + "."
	}

	return domain
}

// splitSRVName parses a DNS-standard SRV name _service._proto.base. into its components.
func splitSRVName(name string) (service, protocol, base string, ok bool) {
	n := strings.TrimSuffix(name, ".")
	labels := strings.SplitN(n, ".", 3)
	if len(labels) < 3 || !strings.HasPrefix(labels[0], "_") || !strings.HasPrefix(labels[1], "_") {
		return "", "", "", false
	}
	service = strings.TrimPrefix(labels[0], "_")
	protocol = strings.ToUpper(strings.TrimPrefix(labels[1], "_"))
	base = normalizeDomainName(labels[2])
	if service == "" || protocol == "" || base == "" {
		return "", "", "", false
	}
	return service, protocol, base, true
}

// normalizeSRVNameFields reconciles the SRV record's name and service/protocol
// with the API's behavior: the server itself adds the `_service._proto.`
// prefix to the supplied name, so the BASE name must be sent. If the name is
// given in the DNS-standard full form (_sip._tcp.zone.), the prefix is
// stripped and service/protocol are derived from it (if not explicitly set).
// Without this, resending the full name would double the prefix:
// _sip._tcp._sip._tcp.zone.
func normalizeSRVNameFields(name string, service *string, protocol **entities.Protocol) string {
	svc, proto, base, ok := splitSRVName(name)
	if !ok {
		return name
	}
	if *service == "" {
		*service = svc
	}
	if *protocol == nil {
		p := entities.Protocol(proto)
		*protocol = &p
	}
	return base
}

// unquoteTXT strips one pair of surrounding quotes: the API stores TXT values
// in presentation form ("..."), which meant the value sent and the value read
// did not match. Applied to all read paths.
func unquoteTXT(s string) string {
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return s[1 : len(s)-1]
	}
	return s
}

// normalizeRecordRead brings a read record into a form symmetric with what
// was written: TXT without surrounding quotes.
func normalizeRecordRead(r *entities.DNSRecord) {
	if r == nil {
		return
	}
	if r.Type == entities.RecordTypeTXT {
		r.Text = unquoteTXT(r.Text)
	}
}

// GetDomains retrieves all domains in the project
func (c *CloudClient) GetDomains(ctx context.Context) ([]*entities.Domain, error) {
	req, err := c.newRequest(ctx, http.MethodGet, domainsBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list domains request: %w", err)
	}

	var resp ListDomainsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}

	for _, d := range resp.Domains {
		for i := range d.Records {
			normalizeRecordRead(&d.Records[i])
		}
	}

	return resp.Domains, nil
}

// GetDomain retrieves a specific domain by name
func (c *CloudClient) GetDomain(ctx context.Context, domainName string) (*entities.Domain, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name is required")
	}

	domainName = normalizeDomainName(domainName)

	path := buildDomainPath(domainName)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for domain %s: %w", domainName, err)
	}

	var resp GetDomainResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get domain %s: %w", domainName, err)
	}

	if resp.Domain == nil {
		return nil, fmt.Errorf("domain %s not found in response: %w", domainName, ErrNotFound)
	}

	for i := range resp.Domain.Records {
		normalizeRecordRead(&resp.Domain.Records[i])
	}

	return resp.Domain, nil
}

// CreateDomain creates a new domain and returns a task ID
func (c *CloudClient) CreateDomain(ctx context.Context, req *entities.CreateDomainRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create domain request is required")
	}
	req.Name = normalizeDomainName(req.Name)
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create domain request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, domainsBaseURL, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create domain request: %w", err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create domain: %w", err)
	}

	return &task, nil
}

// CreateDomainAndWait creates a domain and waits for completion
func (c *CloudClient) CreateDomainAndWait(ctx context.Context, req *entities.CreateDomainRequest) (*entities.Domain, error) {
	task, err := c.CreateDomain(ctx, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for domain creation: %w", err)
	}

	// Get the created domain
	return c.GetDomain(ctx, req.Name)
}

// DeleteDomain deletes a domain.
//
// Deletion is asynchronous: 200 OK means the deletion was accepted, but the
// zone still exists for some time. If you need to wait for actual deletion,
// use DeleteDomainAndWait.
func (c *CloudClient) DeleteDomain(ctx context.Context, domainName string) error {
	if domainName == "" {
		return fmt.Errorf("domain name is required")
	}

	domainName = normalizeDomainName(domainName)

	path := buildDomainPath(domainName)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for domain %s: %w", domainName, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete domain %s: %w", domainName, err)
	}

	return nil
}

// DeleteDomainAndWait deletes a domain and waits until it is actually gone.
// DELETE does not return a task_id, so waiting is implemented by polling
// GetDomain until 404 (the interval/timeout come from the client
// configuration).
func (c *CloudClient) DeleteDomainAndWait(ctx context.Context, domainName string) error {
	if err := c.DeleteDomain(ctx, domainName); err != nil {
		return err
	}

	return c.waitGone(ctx, fmt.Sprintf("domain %s", domainName), func(ctx context.Context) error {
		_, err := c.GetDomain(ctx, domainName)
		return err
	})
}

// GetDomainRecords retrieves all records for a specific domain
func (c *CloudClient) GetDomainRecords(ctx context.Context, domainName string) ([]entities.DNSRecord, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name is required")
	}

	domainName = normalizeDomainName(domainName)

	path := buildDomainPath(domainName, recordsPath)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list records request for domain %s: %w", domainName, err)
	}

	var resp ListRecordsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list records for domain %s: %w", domainName, err)
	}

	for i := range resp.Records {
		normalizeRecordRead(&resp.Records[i])
	}

	return resp.Records, nil
}

// GetDomainRecord retrieves a specific DNS record
func (c *CloudClient) GetDomainRecord(ctx context.Context, domainName string, recordID int) (*entities.DNSRecord, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name is required")
	}
	if recordID <= 0 {
		return nil, fmt.Errorf("record ID must be greater than 0")
	}

	domainName = normalizeDomainName(domainName)

	path := buildDomainPath(domainName, recordsPath, strconv.Itoa(recordID))
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for record %d in domain %s: %w", recordID, domainName, err)
	}

	var resp GetRecordResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get record %d in domain %s: %w", recordID, domainName, err)
	}

	if resp.Record == nil {
		return nil, fmt.Errorf("record %d not found in domain %s: %w", recordID, domainName, ErrNotFound)
	}

	normalizeRecordRead(resp.Record)

	return resp.Record, nil
}

// CreateDomainRecord creates a new DNS record and returns a task ID.
//
// The record name can be given in DNS-standard full form; for SRV records
// the full name (_sip._tcp.zone.) is automatically split into a base name +
// service/protocol — the server itself adds the `_service._proto.` prefix,
// and without normalization the name would end up doubled.
func (c *CloudClient) CreateDomainRecord(ctx context.Context, domainName string, req *entities.CreateRecordRequest) (*TaskID, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name is required")
	}
	if req == nil {
		return nil, fmt.Errorf("create record request is required")
	}

	domainName = normalizeDomainName(domainName)
	req.Name = normalizeDomainName(req.Name)
	if req.Type == entities.RecordTypeSRV {
		req.Name = normalizeSRVNameFields(req.Name, &req.Service, &req.Protocol)
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create record request: %w", err)
	}

	path := buildDomainPath(domainName, recordsPath)
	httpReq, err := c.newRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create record request for domain %s: %w", domainName, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to create record in domain %s: %w", domainName, err)
	}

	return &task, nil
}

// CreateDomainRecordAndWait creates a DNS record and waits for completion
func (c *CloudClient) CreateDomainRecordAndWait(ctx context.Context, domainName string, req *entities.CreateRecordRequest) (*entities.DNSRecord, error) {
	task, err := c.CreateDomainRecord(ctx, domainName, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	completedTask, err := c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for record creation: %w", err)
	}

	// Parse record ID from task result
	if completedTask.RecordID == 0 {
		return nil, fmt.Errorf("record ID not found in task result")
	}

	// Get the created record
	return c.GetDomainRecord(ctx, domainName, completedTask.RecordID)
}

// UpdateDomainRecord updates an existing DNS record and returns a task ID.
//
// As with creation, a full SRV name is normalized to its base name (the
// server adds the `_service._proto.` prefix on PUT as well).
func (c *CloudClient) UpdateDomainRecord(ctx context.Context, domainName string, recordID int, req *entities.UpdateRecordRequest) (*TaskID, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name is required")
	}
	if recordID <= 0 {
		return nil, fmt.Errorf("record ID must be greater than 0")
	}
	if req == nil {
		return nil, fmt.Errorf("update record request is required")
	}

	domainName = normalizeDomainName(domainName)
	req.Name = normalizeDomainName(req.Name)
	if req.Type == entities.RecordTypeSRV {
		req.Name = normalizeSRVNameFields(req.Name, &req.Service, &req.Protocol)
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid update record request: %w", err)
	}

	path := buildDomainPath(domainName, recordsPath, strconv.Itoa(recordID))
	httpReq, err := c.newRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request for record %d in domain %s: %w", recordID, domainName, err)
	}

	var task TaskID
	if err := c.doJSON(httpReq, &task); err != nil {
		return nil, fmt.Errorf("failed to update record %d in domain %s: %w", recordID, domainName, err)
	}

	return &task, nil
}

// UpdateDomainRecordAndWait updates a DNS record and waits for completion
func (c *CloudClient) UpdateDomainRecordAndWait(ctx context.Context, domainName string, recordID int, req *entities.UpdateRecordRequest) (*entities.DNSRecord, error) {
	task, err := c.UpdateDomainRecord(ctx, domainName, recordID, req)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for record update: %w", err)
	}

	// Get the updated record
	return c.GetDomainRecord(ctx, domainName, recordID)
}

// DeleteDomainRecord deletes a DNS record.
//
// Deletion is asynchronous: 200 OK means the deletion was accepted, but the
// record remains in the zone for about 10-20 more seconds. If you need to
// wait for actual deletion, use DeleteDomainRecordAndWait.
func (c *CloudClient) DeleteDomainRecord(ctx context.Context, domainName string, recordID int) error {
	if domainName == "" {
		return fmt.Errorf("domain name is required")
	}
	if recordID <= 0 {
		return fmt.Errorf("record ID must be greater than 0")
	}

	domainName = normalizeDomainName(domainName)

	path := buildDomainPath(domainName, recordsPath, strconv.Itoa(recordID))
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for record %d in domain %s: %w", recordID, domainName, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete record %d in domain %s: %w", recordID, domainName, err)
	}

	return nil
}

// DeleteDomainRecordAndWait deletes a DNS record and waits until it is actually
// gone. DELETE does not return a task_id, so waiting is implemented by
// polling GetDomainRecord until 404 (the interval/timeout come from the
// client configuration).
func (c *CloudClient) DeleteDomainRecordAndWait(ctx context.Context, domainName string, recordID int) error {
	if err := c.DeleteDomainRecord(ctx, domainName, recordID); err != nil {
		return err
	}

	return c.waitGone(ctx, fmt.Sprintf("record %d in domain %s", recordID, domainName), func(ctx context.Context) error {
		_, err := c.GetDomainRecord(ctx, domainName, recordID)
		return err
	})
}

// waitGone polls the get function until it returns 404 (the object is deleted).
func (c *CloudClient) waitGone(ctx context.Context, what string, get func(context.Context) error) error {
	pollingCtx, cancel := context.WithTimeout(ctx, c.config.PollingTimeout)
	defer cancel()

	ticker := time.NewTicker(c.config.PollingInterval)
	defer ticker.Stop()

	for {
		err := get(pollingCtx)
		if IsNotFound(err) {
			return nil
		}
		if err != nil {
			c.logger.Debug("waiting for %s deletion: %v", what, err)
		}

		select {
		case <-pollingCtx.Done():
			return fmt.Errorf("%s was not deleted within %v: %w", what, c.config.PollingTimeout, pollingCtx.Err())
		case <-ticker.C:
		}
	}
}
