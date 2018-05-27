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

func createDNSRecord(domain string, recordName string, recordType string, recordValue string) {
	fmt.Println("Creating record of type", recordType, recordName, recordValue)
	service := route53Service()
	hostedZonesList, err := service.ListHostedZones(&route53.ListHostedZonesInput{})
	dieOnError(err, "Failed to list hosted zones")

	hostedZoneID := ""

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
	fmt.Println("hostedZoneId=", hostedZoneID)

	// TODO: this might work, but needs to be tested
	_, err = service.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &hostedZoneID,
		ChangeBatch: &route53.ChangeBatch{
			Comment: aws.String("ACM Validation Records"),
			Changes: []*route53.Change{
				{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: &recordName,
						Type: &recordType,
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: &recordValue,
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		awserror := err.(awserr.Error)

		fmt.Println("code=", awserror.Code())
	}
	// TODO: Ok, so sometimes we get "invalid reqest" but it's still created?
	dieOnError(err, "Failed to create dns record")
	fmt.Println("Created dns record")
}
