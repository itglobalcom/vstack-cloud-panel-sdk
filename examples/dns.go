package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func runDNSExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== DNS Operations Example ===")

	// Get list of all domains
	fmt.Println("=== Getting domain list ===")
	domains, err := client.GetDomains(ctx)
	if err != nil {
		log.Fatalf("Failed to get domain list: %v", err)
	}
	fmt.Printf("Found %d domains\n", len(domains))
	for _, d := range domains {
		fmt.Printf("  - %s (delegated: %v, records: %d)\n", d.Name, d.IsDelegated, len(d.Records))
	}
	fmt.Println()

	// Create new domain
	fmt.Println("=== Creating new domain ===")
	domainName := "example-test.com"
	domain, err := client.CreateDomainAndWait(ctx, &entities.CreateDomainRequest{
		Name: domainName,
	})
	if err != nil {
		log.Fatalf("[FATAL] Failed to create domain: %v", err)
	}
	fmt.Printf("Domain created: %s (delegated: %v)\n\n", domain.Name, domain.IsDelegated)

	// Get domain details
	fmt.Println("=== Getting domain details ===")
	domain, err = client.GetDomain(ctx, domainName)
	if err != nil {
		log.Fatalf("Failed to get domain: %v", err)
	}
	fmt.Printf("Domain: %s\n", domain.Name)
	fmt.Printf("Delegated: %v\n", domain.IsDelegated)
	fmt.Printf("Records count: %d\n\n", len(domain.Records))

	// Create A record
	fmt.Println("=== Creating A record ===")
	aRecord, err := client.CreateDomainRecordAndWait(ctx, domainName, &entities.CreateRecordRequest{
		Name: "www." + domainName + ".",
		Type: entities.RecordTypeA,
		IP:   "93.184.216.34",
		TTL:  entities.TTL1Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create A record: %v", err)
	}
	fmt.Printf("A record created: ID=%d, Name=%s, IP=%s, TTL=%s\n\n",
		aRecord.ID, aRecord.Name, aRecord.IP, aRecord.TTL)

	// Create AAAA record
	fmt.Println("=== Creating AAAA record ===")
	aaaaRecord, err := client.CreateDomainRecordAndWait(ctx, domainName, &entities.CreateRecordRequest{
		Name: domainName + ".",
		Type: entities.RecordTypeAAAA,
		IP:   "2606:2800:220:1:248:1893:25c8:1946",
		TTL:  entities.TTL1Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create AAAA record: %v", err)
	}
	fmt.Printf("AAAA record created: ID=%d, IP=%s\n\n", aaaaRecord.ID, aaaaRecord.IP)

	// Create MX record
	fmt.Println("=== Creating MX record ===")
	mxPriority := 10
	mxRecord, err := client.CreateDomainRecordAndWait(ctx, domainName, &entities.CreateRecordRequest{
		Name:     domainName + ".",
		Type:     entities.RecordTypeMX,
		MailHost: "mail." + domainName + ".",
		Priority: &mxPriority,
		TTL:      entities.TTL1Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create MX record: %v", err)
	}
	fmt.Printf("MX record created: ID=%d, MailHost=%s, Priority=%d\n\n",
		mxRecord.ID, mxRecord.MailHost, *mxRecord.Priority)

	// Create TXT record
	fmt.Println("=== Creating TXT record ===")
	txtRecord, err := client.CreateDomainRecordAndWait(ctx, domainName, &entities.CreateRecordRequest{
		Name: domainName + ".",
		Type: entities.RecordTypeTXT,
		Text: "v=spf1 -all",
		TTL:  entities.TTL1Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create TXT record: %v", err)
	}
	fmt.Printf("TXT record created: ID=%d, Text=%s\n\n", txtRecord.ID, txtRecord.Text)

	// Create SRV record: DNS-standard full name (_service._proto.zone.) — the
	// SDK derives service/protocol from it itself and sends the base name
	// (the server adds the prefix on its own).
	fmt.Println("=== Creating SRV record ===")
	srvPriority := 10
	srvWeight := 60
	srvPort := 5060
	srvRecord, err := client.CreateDomainRecordAndWait(ctx, domainName, &entities.CreateRecordRequest{
		Name:     "_sip._tcp." + domainName + ".",
		Type:     entities.RecordTypeSRV,
		Priority: &srvPriority,
		Weight:   &srvWeight,
		Port:     &srvPort,
		Target:   "sipserver." + domainName + ".",
		TTL:      entities.TTL30Minutes,
	})
	if err != nil {
		log.Fatalf("Failed to create SRV record: %v", err)
	}
	fmt.Printf("SRV record created: ID=%d, Service=%s, Protocol=%s, Target=%s\n\n",
		srvRecord.ID, srvRecord.Service, string(*srvRecord.Protocol), srvRecord.Target)

	// Get all records for domain
	fmt.Println("=== Getting all domain records ===")
	records, err := client.GetDomainRecords(ctx, domainName)
	if err != nil {
		log.Fatalf("Failed to get domain records: %v", err)
	}
	fmt.Printf("Total records: %d\n", len(records))
	for _, rec := range records {
		fmt.Printf("  - ID=%d, Type=%s, Name=%s, TTL=%s\n",
			rec.ID, rec.Type, rec.Name, rec.TTL)
	}
	fmt.Println()

	// Get specific record
	fmt.Println("=== Getting specific record ===")
	record, err := client.GetDomainRecord(ctx, domainName, aRecord.ID)
	if err != nil {
		log.Fatalf("Failed to get record: %v", err)
	}
	fmt.Printf("Record details: ID=%d, Type=%s, Name=%s, IP=%s\n\n",
		record.ID, record.Type, record.Name, record.IP)

	// Update A record
	fmt.Println("=== Updating A record ===")
	updatedRecord, err := client.UpdateDomainRecordAndWait(ctx, domainName, aRecord.ID, &entities.UpdateRecordRequest{
		IP:   "93.184.216.35",
		TTL:  entities.TTL2Hours,
		Type: entities.RecordTypeA,
		Name: "www." + domainName + ".",
	})
	if err != nil {
		log.Fatalf("Failed to update record: %v", err)
	}
	fmt.Printf("Record updated: ID=%d, IP=%s, TTL=%s\n\n",
		updatedRecord.ID, updatedRecord.IP, updatedRecord.TTL)

	// Delete specific records
	fmt.Println("=== Deleting records ===")

	recordsToDelete := []struct {
		ID   int
		Type string
	}{
		{aRecord.ID, "A"},
		{mxRecord.ID, "MX"},
		{txtRecord.ID, "TXT"},
		{srvRecord.ID, "SRV"},
	}

	for _, rec := range recordsToDelete {
		// Deletion is asynchronous — AndWait waits for the record to actually disappear.
		err = client.DeleteDomainRecordAndWait(ctx, domainName, rec.ID)
		if err != nil {
			log.Printf("Warning: Failed to delete %s record %d: %v", rec.Type, rec.ID, err)
		} else {
			fmt.Printf("%s record (ID=%d) deleted\n", rec.Type, rec.ID)
		}
	}
	fmt.Println()

	// Get remaining records
	fmt.Println("=== Getting remaining records ===")
	remainingRecords, err := client.GetDomainRecords(ctx, domainName)
	if err != nil {
		log.Fatalf("Failed to get remaining records: %v", err)
	}
	fmt.Printf("Remaining records: %d\n", len(remainingRecords))
	for _, rec := range remainingRecords {
		fmt.Printf("  - ID=%d, Type=%s, Name=%s\n", rec.ID, rec.Type, rec.Name)
	}
	fmt.Println()

	// Delete domain (AndWait — wait for the zone to actually be deleted)
	fmt.Println("=== Deleting domain ===")
	err = client.DeleteDomainAndWait(ctx, domainName)
	if err != nil {
		log.Fatalf("Failed to delete domain: %v", err)
	}
	fmt.Printf("Domain %s deleted successfully\n\n", domainName)

	fmt.Println("=== DNS operations completed ===")
}
