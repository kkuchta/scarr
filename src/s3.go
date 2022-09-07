package main

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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

func ensureBucketIsWebsite(bucketName string, region string, redirect string) {
	service := s3Service(region)
	_, err := service.GetBucketWebsite(&s3.GetBucketWebsiteInput{Bucket: &bucketName})
	if err != nil {
		awsError := err.(awserr.Error)
		if awsError.Code() == "NoSuchWebsiteConfiguration" {
			log("Making S3 bucket website...")
			indexFile := "index.html"
			if redirect == "" {
				_, err = service.PutBucketWebsite(&s3.PutBucketWebsiteInput{
					Bucket: &bucketName,
					WebsiteConfiguration: &s3.WebsiteConfiguration{
						IndexDocument: &s3.IndexDocument{Suffix: &indexFile},
					},
				})
			} else {

				_, err = service.PutBucketWebsite(&s3.PutBucketWebsiteInput{
					Bucket: &bucketName,
					WebsiteConfiguration: &s3.WebsiteConfiguration{
						RedirectAllRequestsTo: &s3.RedirectAllRequestsTo{
							HostName: &redirect,
						},
					},
				})
			}

			dieOnError(err, "Failed to update s3 bucket website config")
			logln(" done")
		} else {
			dieOnError(err, "Failed to get bucket website config")
		}
	} else {
		logln("Bucket correctly configured for website")
	}
}

func createBucket(bucketName string, region string) {
	service := s3Service(region)

	input := s3.CreateBucketInput{
		Bucket: &bucketName,
	}

	// Apparently the aws api treats us-east-1 as the default for s3 buckets *and
	// throws an error* if you try to specify us-east-1 as the region.  Wtf.
	if region != "us-east-1" {
		input.CreateBucketConfiguration = &s3.CreateBucketConfiguration{
			LocationConstraint: &region,
		}
	}

	_, err := service.CreateBucket(&input)
	dieOnError(err, "Failed to create bucket")
}

func s3Sync(region string, bucket string, configuredExclude *[]string) []string {
	service := s3ManagerService(region)

	fileList := []string{}
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		for _, exclude := range *configuredExclude {
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
		logf("walk error [%v]\n", err)
	}

	// TODO: detect differences and actually sync, rather than just overwriting everything
	for _, filename := range fileList {
		file, fileErr := os.Open(filename)
		dieOnError(fileErr, "Failed to open file")

		ext := filepath.Ext(filename)

		contentType := ""

		// Detect content type from the extension
		switch ext {
		case ".htm", ".html":
			contentType = "text/html"
		case ".css":
			contentType = "text/css"
		case ".js":
			contentType = "application/javascript"
		default:
			contentType = mime.TypeByExtension(ext)
		}

		// If we can't figure out content type from the extension, try DetectContentType
		if contentType == "" {
			// Grab the first 512 bytes to detect the content type
			buffer := make([]byte, 512)
			_, err = file.Read(buffer)
			dieOnError(err, "Failed reading start of file to detect content type for "+filename)
			// Reset the read pointer if necessary.
			file.Seek(0, 0)
			contentType = http.DetectContentType(buffer)
		}

		logln("Uploading ", filename, " to ", bucket)
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
