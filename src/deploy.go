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

type configType struct {
	Domain string `yaml:"domain"`
	Name   string `yaml:"name"`
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
			if confirm("Register that domain?") {
				registerDomain(config.Domain)
			}
		} else {
			fmt.Println(`
Unfortunately it's not available to register either.  If you own that domain
through a different registrar, transfer it to route53.  Alternately, use
both --skip-dns and --skip-domain to bypass this (you'll have to manage
your own domain + dns setup then)(//TODO: implement those flags)`)
			os.Exit(1)
		}
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
