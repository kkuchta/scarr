package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"os"
)

func amcService() *acm.ACM {
	// ACM needs to do stuff in us-east-1 for cloudfront to work
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")}))
	return acm.New(sess)
}

func getAcmCertificateARN(domain string) *string {
	service := amcService()
	listResult, err := service.ListCertificates(&acm.ListCertificatesInput{})
	dieOnError(err, "Failed to load acm certificates")

	for _, certSummary := range listResult.CertificateSummaryList {
		if *certSummary.DomainName == domain {
			fmt.Println("Found cert:", certSummary)
			return certSummary.CertificateArn
		}
	}

	return nil
}

func createACMCertificate(domain string) *string {
	service := amcService()
	requestResult, err := service.RequestCertificate(&acm.RequestCertificateInput{
		DomainName:              &domain,
		SubjectAlternativeNames: aws.StringSlice([]string{"*." + domain}),
		ValidationMethod:        aws.String("DNS"),
	})
	dieOnError(err, "Failed to request ACM certificate")
	fmt.Println("requestResult=", requestResult)
	setACMDNS(*requestResult.CertificateArn, domain)

	// TODO: wait until the acm cert lists as validated (might need to re-trigger something?)
	return requestResult.CertificateArn
}

func setACMDNS(certificateARN string, domain string) {
	service := amcService()
	describeResult, err := service.DescribeCertificate(&acm.DescribeCertificateInput{
		CertificateArn: &certificateARN,
	})
	dieOnError(err, "Failed to describe ACM certificate")
	fmt.Println("Got detailed cert info")
	domainValidation := describeResult.Certificate.DomainValidationOptions[0]
	// TODO: getting an error here on first run.  Apparently domainValidation.ValidationStatus is nil?
	// Not sure what I should check at this point then.
	if *domainValidation.ValidationStatus == "PENDING_VALIDATION" {
		fmt.Println("One of the cert's validation thingys are still waiting on dns.  Creating...")
		dns := domainValidation.ResourceRecord
		createDNSRecord(domain, *dns.Name, *dns.Type, *dns.Value)
	} else if *domainValidation.ValidationStatus == "FAILED" {
		fmt.Println("Err!  Cert validation failed!")
		os.Exit(1)
	} else {
		fmt.Println("Certificate already validated")
	}

}
