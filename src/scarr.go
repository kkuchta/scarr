package main

import (
	"flag"
	"fmt"
	"os"
	// "github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/session"
	// "github.com/aws/aws-sdk-go/service/s3"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func getUsage() string {
	return `
A tool for generating and updating flat-file sites on AWS.  Registers domain names,
creates TLS certificates, makes S3 buckets, configures cloudfront distributions and
syncs files.

Available commands:
	init		# Generates a new scarr.yml file
	deploy		# Sets up infrastructure + syncs files to it
	version		# Print version
	
Use "scarr <command> -h" for more information.

Scarr requires an AWS IAM user with appropriate permissions.  You can set this user
using one of these two patterns:

	$ AWS_PROFILE=some_profile scarr deploy -args
	$ AWS_ACCESS_KEY_ID=some_id AWS_SECRET_ACCESS_KEY=some_key scarr deploy -args

AWS_PROFILE names a profile set up in an ~/.aws/credentials file as described here:

	https://docs.aws.amazon.com/cli/latest/userguide/cli-config-files.html

Either way, the IAM user you connect should have the following permissions:

{
	"Effect": "Allow",
	"Action": [
		"route53:CreateHostedZone",
		"s3:GetBucketWebsite",
		"route53:ListHostedZones",
		"cloudfront:GetInvalidation",
		"route53:ChangeResourceRecordSets",
		"s3:CreateBucket",
		"s3:ListBucket",
		"cloudfront:CreateDistribution",
		"route53domains:GetDomainDetail",
		"s3:GetBucketAcl",
		"cloudfront:CreateInvalidation",
		"route53domains:GetOperationDetail",
		"s3:PutObject",
		"s3:GetObject",
		"route53domains:CheckDomainAvailability",
		"s3:PutBucketWebsite",
		"acm:DescribeCertificate",
		"acm:RequestCertificate",
		"route53domains:RegisterDomain",
		"cloudfront:ListDistributions",
		"route53:ListResourceRecordSets",
		"s3:PutBucketAcl",
		"acm:ListCertificates",
		"s3:PutObjectAcl"
	],
	"Resource": "*"
}
	`
}

// 0 = silent
// 1 = normal
// todo: verbose/debug
var logLevel int

func logln(msgs ...interface{}) {
	if logLevel == 1 {
		fmt.Println(msgs...)
	}
}
func log(msgs ...interface{}) {
	if logLevel == 1 {
		fmt.Print(msgs...)
	}
}
func logf(msg string, rest ...interface{}) {
	if logLevel == 1 {
		fmt.Printf(msg, rest...)
	}
}

func printVersion() {
	fmt.Println("Scarr " + getVersion())
}

func main() {
	initCommand := flag.NewFlagSet("init", flag.ExitOnError)
	deployCommand := flag.NewFlagSet("deploy", flag.ExitOnError)

	domainPtr := initCommand.String("domain", "", "The domain this site will live at")
	namePtr := initCommand.String("name", "", "The name of this project")
	regionPtr := initCommand.String("region", "us-west-1", "The aws region this project's resources will live in (eg us-west-1)")
	redirectPtr := initCommand.String("redirect", "", "The URL to redirect the domain to.")

	skipSetupPtr := deployCommand.Bool("skip-setup", false, "Assume the infrastructure is all set up and just do the file upload + cache invalidations.")
	autoRegisterPtr := deployCommand.Bool("auto-register", false, "Register the domain name without prompting if necessary and available")
	silentDeployPtr := deployCommand.Bool("silent", false, "Limits stdout to errors and user-input prompts.  Run with -auto-register or use an existing domain name to avoid a registration prompt")

	if len(os.Args) < 2 {
		fmt.Println("Missing command")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "-h":
		fmt.Println(getUsage())
	case "-help":
		fmt.Println(getUsage())
	case "--help":
		fmt.Println(getUsage())
	case "init":
		initCommand.Parse(os.Args[2:])
	case "deploy":
		deployCommand.Parse(os.Args[2:])
	case "version":
		printVersion()
	case "-version":
		printVersion()
	case "--version":
		printVersion()
	case "-v":
		printVersion()
	default:
		fmt.Println("Unknown command ", command)
		flag.PrintDefaults()
		os.Exit(1)
	}
	logLevel = 1

	if initCommand.Parsed() {
		runInit(*domainPtr, *namePtr, *regionPtr, *redirectPtr)
		// fmt.Println("init parsed", *domainPtr, *namePtr, *regionPtr)
	} else if deployCommand.Parsed() {
		if *silentDeployPtr {
			logLevel = 0
		}
		runDeploy(*skipSetupPtr, *autoRegisterPtr)
	}
}
