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

// Returns cloudfrontDomain, distId
func getCloudfront(s3Domain string) (*string, *string) {
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
			if *origin.DomainName == s3Domain {
				// s3Url looks like:
				// voyage-found.s3-website-us-west-1.amazonaws.com
				return dist.DomainName, dist.Id
			}
		}
	}
	return nil, nil
}

func createCloudFront(s3Domain string, bucketName string, certificateArn string, domain string) *string {

	// Taking a break from this function to go set up ACM, since we'll need that ID
	service := cloudFrontService()
	originID := "S3-" + bucketName
	// s3Domain := bucketName + ".s3.amazonaws.com"

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

	// Custom-style origin.
	origin := cloudfront.Origin{
		CustomOriginConfig: &cloudfront.CustomOriginConfig{
			HTTPPort:             aws.Int64(80),
			HTTPSPort:            aws.Int64(443),
			OriginProtocolPolicy: aws.String("http-only"),
		},
		DomainName: &s3Domain,
		Id:         &originID,
	}
	// S3-style origin
	// origin := cloudfront.Origin{
	// 	S3OriginConfig: &cloudfront.S3OriginConfig{
	// 		// Empty for now.  Allows people to access s3 resources directly (which we don't care about)
	// 		OriginAccessIdentity: aws.String(""),
	// 	},
	// 	DomainName: &s3Domain,
	// 	Id:         &originID,
	// }

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

	fmt.Print("Waiting for distribution to finish (20-40 minutes)...")
	service.WaitUntilDistributionDeployed(&cloudfront.GetDistributionInput{
		Id: createResult.Distribution.Id,
	})
	fmt.Println(" done")
	return createResult.Distribution.DomainName
}

func createCloudfrontInvalidation(s3Url string, paths []string) {
	_, distributionID := getCloudfront(s3Url)
	service := cloudFrontService()
	callerReference := time.Now().Format(time.RFC850)
	fmt.Print("Invalidating cache...")
	invalidationResult, err := service.CreateInvalidation(&cloudfront.CreateInvalidationInput{
		DistributionId: distributionID,
		InvalidationBatch: &cloudfront.InvalidationBatch{
			CallerReference: &callerReference,
			Paths: &cloudfront.Paths{
				Items:    aws.StringSlice(paths),
				Quantity: aws.Int64(int64(len(paths))),
			},
		},
	})
	dieOnError(err, "Failed to create Invalidation")

	fmt.Print("waiting (5-10 minutes)...")
	service.WaitUntilInvalidationCompleted(&cloudfront.GetInvalidationInput{
		DistributionId: distributionID,
		Id:             invalidationResult.Invalidation.Id,
	})
	fmt.Println(" done")
}
