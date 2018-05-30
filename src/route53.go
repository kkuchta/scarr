package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53domains"
	"os"
	"strings"
	"time"
)

func route53DomainsService() *route53domains.Route53Domains {
	// Route53 only has the one domain, so hardcode to us east
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")}))
	return route53domains.New(sess)
}
func route53Service() *route53.Route53 {
	// Route53 only has the one domain, so hardcode to us east
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")}))
	return route53.New(sess)
}

func registerDomain(domain string, contactDetails contactDetailsType) {
	contact := route53domains.ContactDetail{
		AddressLine1: &contactDetails.Address1,
		AddressLine2: &contactDetails.Address2,
		City:         &contactDetails.City,
		ContactType:  &contactDetails.ContactType,
		CountryCode:  &contactDetails.CountryCode,
		Email:        &contactDetails.Email,
		FirstName:    &contactDetails.FirstName,
		LastName:     &contactDetails.LastName,
		PhoneNumber:  &contactDetails.PhoneNumber,
		State:        &contactDetails.State,
		ZipCode:      &contactDetails.ZipCode,
	}
	autoRenew := false
	duration := int64(1)
	input := route53domains.RegisterDomainInput{
		AdminContact:      &contact,
		RegistrantContact: &contact,
		TechContact:       &contact,
		AutoRenew:         &autoRenew,
		DomainName:        &domain,
		DurationInYears:   &duration,
	}
	route53DomainsService := route53DomainsService()
	result, err := route53DomainsService.RegisterDomain(&input)
	dieOnError(err, "Failed to register domain")
	fmt.Println("Registration result = ", result)

	operationInput := route53domains.GetOperationDetailInput{
		OperationId: result.OperationId,
	}
	operationResult, err := route53DomainsService.GetOperationDetail(&operationInput)
	dieOnError(err, "Failed to get registration operation (but probs still registered the domains")
	fmt.Println("Registration Operation result = ", operationResult)
	// TODO: loop around this.  Probably loop while operationResult.status != something
}

func getDomainDetails(domain string) *route53domains.GetDomainDetailOutput {
	route53DomainsService := route53DomainsService()

	// So, apparently the only way to determine if a domain exists in our route53
	// is to try to fetch it and see if we get a specific error.  Well, ok, we could
	// call ListDomains, but then we'd have to paginate through an arbitrarily
	// large list which is just silly.
	input := route53domains.GetDomainDetailInput{DomainName: &domain}
	result, err := route53DomainsService.GetDomainDetail(&input)
	if err != nil {
		if strings.Contains(err.Error(), "Domain "+domain+" not found in") {
			return nil
		}
		dieOnError(err, "error loading domain: ")
	}

	// TODO: handle domain-found result
	//fmt.Println("get domain details result = ", result)
	return result
}

func getDomainAvailability(domain string) bool {
	return getDomainAvailabilityWithRetries(domain, 3)
}

func getDomainAvailabilityWithRetries(domain string, retries int) bool {

	route53DomainsService := route53DomainsService()
	input := route53domains.CheckDomainAvailabilityInput{DomainName: &domain}
	availabilityResult, err := route53DomainsService.CheckDomainAvailability(&input)
	fmt.Println("av result =", availabilityResult)
	dieOnError(err, "error getting domain availability")
	if *availabilityResult.Availability == "AVAILABLE" {
		return true
	}
	if *availabilityResult.Availability == "PENDING" {
		if retries > 0 {
			time.Sleep(time.Second)
			return getDomainAvailabilityWithRetries(domain, retries-1)
		}
	}
	return false
}

func dnsRecordExists(hostedZoneID string, domain string, recordType string) bool {
	service := route53Service()
	result, err := service.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId: &hostedZoneID,
	})
	dieOnError(err, "Failed listing resource record sets")

	for _, recordSet := range result.ResourceRecordSets {
		if *recordSet.Name == domain+"." && *recordSet.Type == recordType {
			fmt.Println("Found alias!")
			return true
		}
	}
	return false
}

func getHostedZone(domain string) string {
	service := route53Service()

	hostedZoneID := ""

	hostedZonesList, err := service.ListHostedZones(&route53.ListHostedZonesInput{})
	dieOnError(err, "Failed to list hosted zones")

	for _, hostedZone := range hostedZonesList.HostedZones {
		if *hostedZone.Name == domain+"." {
			hostedZoneID = *hostedZone.Id
		}
	}

	if hostedZoneID == "" {
		// TODO: we can probably just create the hosted zone in this case
		fmt.Println("Couldn't find hosted zone for domain")
		os.Exit(1)
	}
	return hostedZoneID
}

func createAliasRecord(hostedZoneDomain string, recordName string, cloudfrontDomain string) {
	createDNSRecord(hostedZoneDomain, recordName, "A", nil, &route53.AliasTarget{
		DNSName:              &cloudfrontDomain,
		EvaluateTargetHealth: aws.Bool(false),
		HostedZoneId:         aws.String("Z2FDTNDATAQYW2"),
	})
	// ^ Hardcoded zone ID as specified in aws docs
}

func createDNSRecord(domain string, recordName string, recordType string, recordValue *string, aliasTarget *route53.AliasTarget) {
	// fmt.Println("Creating record of type", recordType, recordName, recordValue)
	service := route53Service()

	hostedZoneID := getHostedZone(domain)

	input := route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &hostedZoneID,
		ChangeBatch: &route53.ChangeBatch{
			Comment: aws.String("Created by scarr.io"),
			Changes: []*route53.Change{
				{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: &recordName,
						Type: &recordType,
					},
				},
			},
		},
	}

	if recordValue != nil {
		input.ChangeBatch.Changes[0].ResourceRecordSet.ResourceRecords = []*route53.ResourceRecord{
			{
				Value: recordValue,
			},
		}
	}

	// If we're setting an alias record, we need this extra input
	if aliasTarget != nil {
		input.ChangeBatch.Changes[0].ResourceRecordSet.AliasTarget = aliasTarget
	} else {
		// Maybe necessary for cert validation dns?
		input.ChangeBatch.Changes[0].ResourceRecordSet.TTL = aws.Int64(300)
	}

	_, err := service.ChangeResourceRecordSets(&input)
	if err != nil {
		awserror := err.(awserr.Error)

		fmt.Println("code=", awserror.Code())
	}

	dieOnError(err, "Failed to create dns record")
}
