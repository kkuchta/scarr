package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"os"
	"time"
)

func cloudFrontService() *cloudfront.CloudFront {
	sess := session.Must(session.NewSession(&aws.Config{}))
	return cloudfront.New(sess)
}

func cloudFrontExists(s3Url string) bool {
	service := cloudFrontService()
	result, err := service.ListDistributions(&cloudfront.ListDistributionsInput{})
	dieOnError(err, "Failed getting distribution list")
	if *result.DistributionList.IsTruncated {
		// If you have over 1k distributions
		fmt.Println("TODO: handle paginated result lists for cloudfront")
		os.Exit(1)
	}

	for _, dist := range result.DistributionList.Items {
		for _, origin := range dist.Origins.Items {
			if *origin.DomainName == s3Url {
				// s3Url looks like:
				// voyage-found.s3-website-us-west-1.amazonaws.com
				return true
			}
		}
	}
	return false
}

func createCloudFront(s3Url string, bucketName string, certificateArn string, domain string) {
	fmt.Println("Creating cloudfront")
	fmt.Println("s3Url=", s3Url)
	fmt.Println("bucketName=", bucketName)
	fmt.Println("certificateArn=", certificateArn)
	fmt.Println("domain=", domain)
	// Taking a break from this function to go set up ACM, since we'll need that ID
	service := cloudFrontService()
	originID := "S3-" + bucketName
	// s3DomainName := bucketName + ".s3.amazonaws.com"

	aliases := cloudfront.Aliases{
		Items:    aws.StringSlice([]string{domain}),
		Quantity: aws.Int64(1),
	}

	defaultCacheBehavior := cloudfront.DefaultCacheBehavior{
		AllowedMethods: &cloudfront.AllowedMethods{
			Items:    aws.StringSlice([]string{"GET", "HEAD"}),
			Quantity: aws.Int64(2),
			CachedMethods: &cloudfront.CachedMethods{
				Items:    aws.StringSlice([]string{"GET", "HEAD"}),
				Quantity: aws.Int64(2),
			},
		},
		Compress: aws.Bool(true),
		ForwardedValues: &cloudfront.ForwardedValues{
			Cookies: &cloudfront.CookiePreference{
				Forward: aws.String("none"),
			},
			QueryString: aws.Bool(false),
		},
		MinTTL:         aws.Int64(0),
		TargetOriginId: &originID,
		TrustedSigners: &cloudfront.TrustedSigners{
			Enabled:  aws.Bool(false),
			Quantity: aws.Int64(0),
		},
		ViewerProtocolPolicy: aws.String("redirect-to-https"),
	}

	s3Domain := bucketName + ".s3.amazonaws.com"

	fmt.Println("s3Domain=", s3Domain)
	origin := cloudfront.Origin{
		// S3OriginConfig: &cloudfront.S3OriginConfig{
		// 	OriginAccessIdentity: aws.String(""),
		// },
		CustomOriginConfig: &cloudfront.CustomOriginConfig{
			HTTPPort:             aws.Int64(80),
			HTTPSPort:            aws.Int64(443),
			OriginProtocolPolicy: aws.String("https-only"),
		},
		DomainName: &s3Domain,
		Id:         &originID,
	}

	origins := cloudfront.Origins{
		Items:    []*cloudfront.Origin{&origin},
		Quantity: aws.Int64(1),
	}

	certificate := cloudfront.ViewerCertificate{
		ACMCertificateArn:      &certificateArn,
		SSLSupportMethod:       aws.String("sni-only"),
		MinimumProtocolVersion: aws.String("TLSv1"),
	}

	callerReference := time.Now().Format(time.RFC850)

	config := cloudfront.DistributionConfig{
		Aliases:              &aliases,
		CallerReference:      &callerReference,
		Comment:              aws.String("Created by Scarr.io"),
		DefaultCacheBehavior: &defaultCacheBehavior,
		CacheBehaviors:       &cloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
		Enabled:              aws.Bool(true),
		CustomErrorResponses: &cloudfront.CustomErrorResponses{Quantity: aws.Int64(0)},
		PriceClass:           aws.String("PriceClass_All"),
		Restrictions: &cloudfront.Restrictions{
			GeoRestriction: &cloudfront.GeoRestriction{
				RestrictionType: aws.String("none"),
				Quantity:        aws.Int64(0),
			},
		},
		Origins:           &origins,
		ViewerCertificate: &certificate,
	}
	createResult, err := service.CreateDistribution(&cloudfront.CreateDistributionInput{DistributionConfig: &config})

	if err != nil {
		awserror := err.(awserr.Error)
		fmt.Println("code=", awserror.Code())
		fmt.Println(awserror.Message())
	}

	dieOnError(err, "Failed to create cloudfront distribution")

	fmt.Println("Waiting for distribution to finish...")
	service.WaitUntilDistributionDeployed(&cloudfront.GetDistributionInput{
		Id: createResult.Distribution.Id,
	})
	fmt.Println("Distribution finished!")
}
