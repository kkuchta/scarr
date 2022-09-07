package main

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
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
	setACMDNS(*requestResult.CertificateArn, domain)

	// TODO: wait until the acm cert lists as validated (might need to re-trigger something?)
	return requestResult.CertificateArn
}

func getCertificateValidation(certificateARN string) *acm.DomainValidation {
	service := amcService()
	describeResult, err := service.DescribeCertificate(&acm.DescribeCertificateInput{
		CertificateArn: &certificateARN,
	})
	dieOnError(err, "Failed to describe ACM certificate")
	return describeResult.Certificate.DomainValidationOptions[0]
}

func setACMDNS(certificateARN string, domain string) {

	var domainValidation *acm.DomainValidation
	// Right after certificate creation, validation status seems to be nil.  Wait a bit.
	for i := 0; i < 5; i++ {
		domainValidation := getCertificateValidation(certificateARN)
		if domainValidation.ValidationStatus != nil {
			break
		} else {
			time.Sleep(5 * time.Second)
		}
	}
	// TODO: read up on go memory management so I don't have to do this again here to avoid segfaults
	domainValidation = getCertificateValidation(certificateARN)

	if *(domainValidation.ValidationStatus) == "PENDING_VALIDATION" {
		log("not yet valid; creating validation dns records...")
		dns := domainValidation.ResourceRecord
		// If the dns record already exists, we're just waiting for validation so don't try to recreate it.
		if !dnsRecordExists(getHostedZone(domain), *dns.Name, *dns.Type) {
			createDNSRecord(domain, *dns.Name, *dns.Type, dns.Value, nil)
		}

		log("waiting for validation (takes up to a few hours - feel free to ctrl-c and restart scarr later)...")
		time.Sleep(5 * time.Second)

		maxTries := 60 * 3
		for i := 0; i < maxTries; i++ {
			if *getCertificateValidation(certificateARN).ValidationStatus != "PENDING_VALIDATION" {
				setACMDNS(certificateARN, domain)
				break
			}
			if i == (maxTries - 1) {
				logln("\nTimed out waiting for ACM certificate to validate.")
				os.Exit(1)
			}
			time.Sleep(60 * time.Second)
		}
	} else if *domainValidation.ValidationStatus == "FAILED" {
		logln("Err!  Cert validation failed!")
		os.Exit(1)
	} else {
		log("Certificate validated")
	}

}
