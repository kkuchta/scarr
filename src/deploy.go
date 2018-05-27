package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type contactDetailsType struct {
	Address1    string `yaml:"address1"`
	Address2    string `yaml:"address2"`
	City        string `yaml:"city"`
	ContactType string `yaml:"contactType"`
	CountryCode string `yaml:"countryCode"`
	Email       string `yaml:"email"`
	FirstName   string `yaml:"firstName"`
	LastName    string `yaml:"lastName"`
	PhoneNumber string `yaml:"phoneNumber"`
	State       string `yaml:"state"`
	ZipCode     string `yaml:"zipCode"`
}

type configType struct {
	Domain        string             `yaml:"domain"`
	Name          string             `yaml:"name"`
	Region        string             `yaml:"region"`
	DomainContact contactDetailsType `yaml:"domainContact"`
}

func dieOnError(err error, message string) {
	if err != nil {
		fmt.Println(message, err)
		os.Exit(1)
	}
}

func confirm(message string) bool {
	fmt.Print(message + " [y/N]")

	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	fmt.Println("")
	dieOnError(err, "Failed reading y/n input")
	return "y" == strings.TrimSpace(strings.ToLower(text))
}

func getConfig() configType {
	yamlFile, err := ioutil.ReadFile("scarr.yml")
	dieOnError(err, "Error reading scarr.yml")
	if err != nil {
		log.Printf("Error reading scarr.yml err: #%v ", err)
	}

	var config configType
	err = yaml.Unmarshal(yamlFile, &config)
	dieOnError(err, "Error parsing scarr.yml")

	return config
}

func ensureDomainRegistered(config configType) {

	domainDetail := getDomainDetails(config.Domain)
	if domainDetail == nil {
		fmt.Println("Couldn't find domain '" + config.Domain + "' in your route53 account.")

		// Not clear if there's a good way to detect this
		// if isRegistering(config.Domain) {
		// 	fmt.Println("Your domain is still registering.  Try again later.")
		// 	return
		// }

		domainAvailability := getDomainAvailability(config.Domain)
		if domainAvailability {
			fmt.Println(`
But it *is* available to register.  For current prices, see the document linked at:
https://aws.amazon.com/route53/pricing/
				`)
			if strings.HasSuffix(config.Domain, ".com") {
				fmt.Println("But as of April 2018, .com TLDs were $12/yr")
			}
			if confirm("Register that domain?") {
				registerDomain(config.Domain, config.DomainContact)
			}
		} else {
			fmt.Println(`
Unfortunately it's not available to register either.  Maybe it's still
registering from the last time you ran scarr?  If so, try again in a few.
If you own that domain through a different registrar, transfer it to
route53.  Alternately, use both --skip-dns and --skip-domain to bypass
this (you'll have to manage your own domain + dns setup then)
(//TODO: implement those flags)`)
			os.Exit(1)
		}
	} else {
		fmt.Println("Domain exists!")
		//fmt.Println("domainDetail=", domainDetail)
	}
}
func ensureS3BucketExists(s3BucketName string, region string) {
	if !bucketExists(s3BucketName, region) {
		fmt.Println("S3 Bucket doesn't exist - creating it.")
		createBucket(s3BucketName, region)
	} else {
		fmt.Println("Bucket exists")
	}

	if !bucketIsWorldReadable(s3BucketName, region) {
		// We could _make_ this bucket world-readable, but that'd be bad if it turns out to have sensitive info in it.
		fmt.Println("Bucket is not world-readable.  You should fix this (or delete the bucket and let us re-create it).")
		os.Exit(1)
	}
	ensureBucketIsWebsite(s3BucketName, region)
}

func ensureACMCertificate(domain string) string {
	certificateArn := getAcmCertificateARN(domain)
	if certificateArn == nil {
		fmt.Println("ACM certificate for this domain doesn't exist - creating one.")
		certificateArn = createACMCertificate(domain)
	} else {
		// Ensure it's DNS is set up
		fmt.Println("Certificate already exists - ensuring it's validated")
		setACMDNS(*certificateArn, domain)
	}
	return *certificateArn
}
func ensureCloudFrontExists(certificateArn string, s3Url string, s3Bucket string, domain string) {
	if !cloudFrontExists(s3Url) {
		fmt.Println("CloudFront distribution does not exist; creating")
		createCloudFront(s3Url, s3Bucket, certificateArn, domain)
	}
}

func runDeploy() {
	config := getConfig()
	s3Bucket := config.Name + "-bucket"
	s3Url := s3Bucket + ".s3-website-" + config.Region + ".amazonaws.com"
	fmt.Println("s3Url=", s3Url)

	ensureDomainRegistered(config)
	certArn := ensureACMCertificate(config.Domain)
	ensureS3BucketExists(s3Bucket, config.Region)
	ensureCloudFrontExists(certArn, s3Url, s3Bucket, config.Domain)

	//changed_files = sync_to_s3(s3Bucket)
	//invalidate(changed_files)

}
