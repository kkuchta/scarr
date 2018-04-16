package main

import (
  "io/ioutil"
  "fmt"
  "log"

  "gopkg.in/yaml.v2"
)

type ConfigType struct {
  Domain string `yaml:"domain"`
  Name string `yaml:"name"`
}

func getConfig() ConfigType {
  yamlFile, err := ioutil.ReadFile("scarr.yml")
  if err != nil {
    log.Printf("Error reading scarr.yml err: #%v ", err)
  }

  var config ConfigType
  err = yaml.Unmarshal(yamlFile, &config)
  if err != nil {
      log.Fatalf("Error parsing scarr.yml: %v", err)
  }

  return config
}

func runDeploy() {
  fmt.Println("deploying for realz")
  config := getConfig()
  fmt.Println("config=", config.Domain)
  //s3Bucket = conf.name + "_bucket"

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
