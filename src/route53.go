package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53domains"
	"strings"
	"time"
)

func route53Service() *route53domains.Route53Domains {
	// Route53 only has the one domain, so hardcode to us east
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")}))
	return route53domains.New(sess)
}

func registerDomain(domain string) {
	contact := route53domains.ContactDetail{
		//todo
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
	route53DomainsService := route53Service()
	result, err := route53DomainsService.RegisterDomain(&input)
	dieOnError(err, "Failed to register domain")
	fmt.Println("Registration result = ", result)
}

func getDomainDetails(domain string) *string {
	route53DomainsService := route53Service()

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
	fmt.Println("result = ", result)
	blah := "foo"
	return &blah
}

func getDomainAvailability(domain string) bool {
	return getDomainAvailabilityWithRetries(domain, 3)
}

func getDomainAvailabilityWithRetries(domain string, retries int) bool {

	route53DomainsService := route53Service()
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
