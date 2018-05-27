package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func s3Service(region string) *s3.S3 {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region)}))
	return s3.New(sess)
}

func bucketExists(bucketName string, region string) bool {
	fmt.Println("checking bucket", bucketName)
	service := s3Service(region)
	result, err := service.HeadBucket(&s3.HeadBucketInput{Bucket: &bucketName})

	if err != nil {
		// If we got an error, it's *probably* just because this bucket needs to be
		// created.  Hopefully.  We might also not have permissions to list buckets!
		fmt.Println("Error heading bucket (probably just need to create it):", err)
		fmt.Println(" head result =", result)
		return false
	}
	fmt.Println("bucket head ersult =", result)
	return true
}

func bucketIsWorldReadable(bucketName string, region string) bool {
	service := s3Service(region)
	aclResult, err := service.GetBucketAcl(&s3.GetBucketAclInput{
		Bucket: &bucketName,
	})
	dieOnError(err, "Error getting bucket ACL")

	// Make sure this bucket is publicly readable
	for _, grant := range aclResult.Grants {
		if *grant.Grantee.Type == "Group" &&
			*grant.Grantee.URI == "http://acs.amazonaws.com/groups/global/AllUsers" &&
			*grant.Permission == "READ" {
			return true
		}
	}
	return false
}

func ensureBucketIsWebsite(bucketName string, region string) {
	service := s3Service(region)
	_, err := service.GetBucketWebsite(&s3.GetBucketWebsiteInput{Bucket: &bucketName})
	if err != nil {
		awsError := err.(awserr.Error)
		if awsError.Code() == "NoSuchWebsiteConfiguration" {
			fmt.Println("S3 bucket not configured for website - fixing")
			indexFile := "index.html"
			_, err = service.PutBucketWebsite(&s3.PutBucketWebsiteInput{
				Bucket: &bucketName,
				WebsiteConfiguration: &s3.WebsiteConfiguration{
					IndexDocument: &s3.IndexDocument{Suffix: &indexFile},
				},
			})
			dieOnError(err, "Failed to update s3 bucket website config")
			fmt.Println("Updated s3 bucket to be a website")

			// TODO create website configuration
		} else {
			dieOnError(err, "Failed to get bucket website config")
		}
	} else {
		fmt.Println("Bucket correctly configured for website")
	}
}

func createBucket(bucketName string, region string) {
	service := s3Service(region)
	fmt.Println("region=", region)

	acl := "public-read"
	input := s3.CreateBucketInput{
		Bucket: &bucketName,
		ACL:    &acl,
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: &region,
		},
	}
	result, err := service.CreateBucket(&input)
	dieOnError(err, "Failed to create bucket")
	fmt.Println("Create bucket result = ", result)
}
