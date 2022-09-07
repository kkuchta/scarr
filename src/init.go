package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"
)

const configTemplateString = `domain: "{{.domain}}"
name: "{{.name}}"
region: "{{.region}}"

# If this is a URL, the bucket will instead redirect all requests to that URL.
redirect: "{{.redirect}}"

# This section's only used if you use scarr to register a domain.  Which fields
# are required depends on what TLD you register.  See 
# https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/domain-register-values-specify.html
# for details.
domainContact:
  address1: 'fillmein'
  address2: ''
  city: 'fillmein'
  contactType: 'PERSON'
  countryCode: 'fillmein'
  email: 'fillmein'
  firstName: 'fillmein'
  lastName: 'fillmein'
  phoneNumber: 'fillmein'
  state: 'fillmein'
  zipCode: 'fillmein'

# A list of regexes to be run against paths in the current directory.  Any file path matching any of these regexes will not be synced to s3
exclude:
  - "scarr\\.yml"
  - "^\\.git"
  - "\\.DS_Store"
`

func generateConfig(domain string, name string, region string, redirect string) string {
	configTemplate := template.Must(template.New("config").Parse(configTemplateString))
	buffer := &bytes.Buffer{}
	data := map[string]interface{}{
		"name":     name,
		"domain":   domain,
		"region":   region,
		"redirect": redirect,
	}
	check(configTemplate.Execute(buffer, data))

	return buffer.String()
}

func writeFile(path string, content string) {
	check(ioutil.WriteFile(path, []byte(content), 0644))
}

func runInit(domain string, name string, region string, redirect string) {
	log("Initializing...")
	err := os.Mkdir(name, 0755)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	config := generateConfig(domain, name, region, redirect)
	writeFile(name+"/scarr.yml", config)
	logln("done")
	logln("You'll need to edit scarr.yml to fill in contact details if you want to use scarr register domain names.")
}
