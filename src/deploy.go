package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
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
	Exclude       []string           `yaml:"exclude"`
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

	var config configType
	err = yaml.Unmarshal(yamlFile, &config)
	dieOnError(err, "Error parsing scarr.yml")

	return config
}

func ensureDomainRegistered(config configType, autoRegister bool) {
	logln("Checking domain %v registration...", config.Domain)

	domainDetail := getDomainDetails(config.Domain)
	if domainDetail == nil {
		logln("\nNot registered in our Route53")

		// Not clear if there's a good way to detect this
		// if isRegistering(config.Domain) {
		// 	fmt.Println("Your domain is still registering.  Try again later.")
		// 	return
		// }

		domainAvailability := getDomainAvailability(config.Domain)
		if domainAvailability {
			logln(`
But it *is* available to register.  For current prices, see the document linked at:
https://aws.amazon.com/route53/pricing/
				`)
			if strings.HasSuffix(config.Domain, ".com") {
				logln("(As of April 2018, .com TLDs were $12/yr)")
			}
			if autoRegister || confirm("Register that domain?") {
				registerDomain(config.Domain, config.DomainContact)
			}
		} else {
			fmt.Println(`
Unfortunately that domain is not available to register.  Maybe it's still
registering from the last time you ran scarr?  If so, try again in a few.
If you own that domain through a different registrar, transfer it to
route53.  Alternately, use both --skip-dns and --skip-domain to bypass
this (you'll have to manage your own domain + dns setup then)
(//TODO: implement those flags)`)
			os.Exit(1)
		}
	} else {
		logln("Looks good!")
	}
}
func ensureS3BucketExists(s3BucketName string, region string) {
	logf("Checking bucket %v...", s3BucketName)
	if !bucketExists(s3BucketName, region) {
		log(" bucket doesn't exist; creating it now...")
		createBucket(s3BucketName, region)
	} else {
		log(" bucket already exists.")
	}

	// if !bucketIsWorldReadable(s3BucketName, region) {
	// 	// We could _make_ this bucket world-readable, but that'd be bad if it turns out to have sensitive info in it.
	// 	logln("\nBucket is not world-readable.  You should fix this (or delete the bucket and let us re-create it).")
	// 	os.Exit(1)
	// }
	logln(" done")
	ensureBucketIsWebsite(s3BucketName, region)
}

func ensureACMCertificate(domain string) string {
	logf("Checking ACM cert for %v...", domain)
	certificateArn := getAcmCertificateARN(domain)
	if certificateArn == nil {
		log("doesn't exist; creating...")
		certificateArn = createACMCertificate(domain)
	} else {
		// Ensure it's DNS is set up
		log("already exists; ensuring it's validated...")
		setACMDNS(*certificateArn, domain)
	}
	logln(" done")
	return *certificateArn
}
func ensureCloudFrontExists(certificateArn string, s3Url string, s3Bucket string, domain string) string {
	cloudfrontDomain, _ := getCloudfront(s3Url)
	if cloudfrontDomain == nil {
		logln("CloudFront distribution does not exist; creating")
		cloudfrontDomain = createCloudFront(s3Url, s3Bucket, certificateArn, domain)
	}
	return *cloudfrontDomain
}
func ensureDomainPointingToCloudfront(cloudfrontDomain string, mainDomain string) {
	hostedZoneID := getHostedZone(mainDomain)
	if dnsRecordExists(hostedZoneID, mainDomain, "A") {
		logln("Domain has a (hopefully-correct) alias already configured")
	} else {
		logln("Creating A-record alias to domain")
		createAliasRecord(mainDomain, mainDomain, cloudfrontDomain)
	}

	// TODO: set up an alias or redirect from www to apex
}

func invalidateCloudfront(s3Domain string, pathsToInvalidate []string) {
	// TODO: actually invalidate what's passed in
	createCloudfrontInvalidation(s3Domain, []string{"/*"})
}

func runDeploy(skipSetup bool, autoRegister bool) {
	logln("Deploying")
	// TODO: implement skipsetup and autoregister
	config := getConfig()
	s3Bucket := config.Name + "-bucket"
	s3Url := s3Bucket + ".s3-website-" + config.Region + ".amazonaws.com"

	if !skipSetup {
		ensureDomainRegistered(config, autoRegister)
		certArn := ensureACMCertificate(config.Domain)
		ensureS3BucketExists(s3Bucket, config.Region)
		cloudfrontDomain := ensureCloudFrontExists(certArn, s3Url, s3Bucket, config.Domain)
		ensureDomainPointingToCloudfront(cloudfrontDomain, config.Domain)
	}

	changedFiles := s3Sync(config.Region, s3Bucket, &config.Exclude)
	invalidateCloudfront(s3Url, changedFiles)
}
