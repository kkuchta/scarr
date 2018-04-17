package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53domains"
)

type configType struct {
	Domain string `yaml:"domain"`
	Name   string `yaml:"name"`
}

func route53Service() *route53domains.Route53Domains {
	// Route53 only has the one domain, so hardcode to us east
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")}))
	return route53domains.New(sess)
}

func getConfig() configType {
	yamlFile, err := ioutil.ReadFile("scarr.yml")
	if err != nil {
		log.Printf("Error reading scarr.yml err: #%v ", err)
	}

	var config configType
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Println("error parsing scarr.yml:", err)
		os.Exit(1)
	}

	return config
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
		fmt.Println("error loading domain: ", err)
		os.Exit(1)
	}

	// TODO: handle domain-found result
	fmt.Println("result = ", result)
	blah := "foo"
	return &blah
}

func getDomainAvailability(domain string) bool {
	route53DomainsService := route53Service()
	input := route53domains.CheckDomainAvailabilityInput{DomainName: &domain}
	availabilityResult, err := route53DomainsService.CheckDomainAvailability(&input)
	fmt.Println("av result =", availabilityResult)
	if err != nil {
		fmt.Println("error getting domain availability: ", err)
		os.Exit(1)
	}
	if *availabilityResult.Availability == "AVAILABLE" {
		return true
	}
	return false
}

func runDeploy() {
	fmt.Println("deploying for realz")
	config := getConfig()
	s3Bucket := config.Name + "_bucket"
	fmt.Println("s3=", s3Bucket)

	domainDetail := getDomainDetails(config.Domain)
	if domainDetail == nil {
		fmt.Println("Couldn't find domain '" + config.Domain + "' in your route53 account.")

		domainAvailability := getDomainAvailability(config.Domain)
		if domainAvailability {
			fmt.Println(`
But it *is* available to register.  For current prices, see the document linked at:
https://aws.amazon.com/route53/pricing/
				`)
			if strings.HasSuffix(config.Domain, ".com") {
				fmt.Println("But as of April 2018, .com TLDs were $12/yr")
			}
		} else {
			fmt.Println(`
Unfortunately it's not available to register either.  If you own that domain
through a different registrar, transfer it to route53.  Alternately, use
both --skip-dns and --skip-domain to bypass this (you'll have to manage
your own domain + dns setup then)(//TODO: implement those flags)`)
			exit(1)
		}
		//reader := bufio.NewReader(os.Stdin)
		//text, _ := reader.ReadString('\n')
	}

	fmt.Println("domainDetail=", domainDetail)
	//dns_settings = getDns(config.domain)
	//ensure_domain_registered(dns_settings, config.domain)
	//ensure_s3_bucket_exists(s3Bucket)
	//changed_files = sync_to_s3(s3Bucket)
	//cloudfront_existed = ensure_cloudfront_exists
	//# Add cloudfront_id to config if created?
	//ensure_ssl_cert_exists_and_is_on_cloudfront
	//if cloudfront_existed
	//invalidate(changed_files)
}
