package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

func s3Service(region string) *s3.S3 {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region)}))
	return s3.New(sess)
}
func s3ManagerService(region string) *s3manager.Uploader {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region)}))
	return s3manager.NewUploader(sess)
}

func bucketExists(bucketName string, region string) bool {
	fmt.Println("checking bucket", bucketName)
	service := s3Service(region)
	_, err := service.HeadBucket(&s3.HeadBucketInput{Bucket: &bucketName})

	if err != nil {
		awsError := err.(awserr.Error)
		if awsError.Code() == "NotFound" {
			return false
		}
		dieOnError(err, "Error HEADing bucket")
	}
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

	acl := "public-read"
	input := s3.CreateBucketInput{
		Bucket: &bucketName,
		ACL:    &acl,
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: &region,
		},
	}
	_, err := service.CreateBucket(&input)
	dieOnError(err, "Failed to create bucket")
}

func s3Sync(region string, bucket string, configuredExclude *[]string) []string {
	service := s3ManagerService(region)
	defaultExclude := []string{
		".git",
		".DS_Store",
	}
	allExclude := append(defaultExclude, *configuredExclude...)

	fileList := []string{}
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		for _, exclude := range allExclude {
			matched, err := regexp.MatchString(exclude, path)
			dieOnError(err, "Invalid exclude regex")
			if matched {
				return nil
			}
		}

		fileList = append(fileList, path)
		return nil
	})
	if err != nil {
		fmt.Printf("walk error [%v]\n", err)
	}

	// TODO: detect differences and actually sync, rather than just overwriting everything
	for _, filename := range fileList {
		file, fileErr := os.Open(filename)
		dieOnError(fileErr, "Failed to open file")

		// Grab the first 512 bytes to detect the content type
		buffer := make([]byte, 512)
		_, err = file.Read(buffer)
		dieOnError(err, "Failed reading start of file to detect content type")
		// Reset the read pointer if necessary.
		file.Seek(0, 0)
		contentType := http.DetectContentType(buffer)
		fmt.Println("Uploading ", filename, " to ", bucket)
		_, uploadErr := service.Upload(&s3manager.UploadInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(filename),
			Body:        file,
			GrantRead:   aws.String("uri=http://acs.amazonaws.com/groups/global/AllUsers"),
			ContentType: &contentType,
		})
		dieOnError(uploadErr, "Failed to upload file")
	}
	return fileList
}
