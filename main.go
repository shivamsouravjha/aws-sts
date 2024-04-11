package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

func main() {
	ctx := context.Background()

	// Create a new session without credentials.
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentialsFromCreds(credentials.Value{
			AccessKeyID:     "<AccessKeyID>",
			SecretAccessKey: "<SecretAccessKey>",
		}),
	}))

	// Create an STS client.
	stsSvc := sts.New(sess)

	// Assume a role.
	assumeRoleInput := &sts.AssumeRoleInput{
		RoleArn:         aws.String("<rolearn>"),
		RoleSessionName: aws.String("<MySession>"),
		DurationSeconds: aws.Int64(3600), // 1 hour
	}
	assumeRoleOutput, err := stsSvc.AssumeRole(assumeRoleInput)
	if err != nil {
		log.Fatalf("Failed to assume role: %v", err)
	}

	// Use the temporary credentials from the assumed role.
	creds := credentials.NewStaticCredentials(
		*assumeRoleOutput.Credentials.AccessKeyId,
		*assumeRoleOutput.Credentials.SecretAccessKey,
		*assumeRoleOutput.Credentials.SessionToken,
	)

	// Create a new AWS session with the temporary credentials.
	sessWithCreds := session.Must(session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String("<location>"), // Set your AWS region
	}))

	// Create a *blob.Bucket using the s3blob package.
	bucket, err := s3blob.OpenBucket(ctx, sessWithCreds, "<bucketName>", nil)
	if err != nil {
		log.Fatalf("Failed to open bucket: %v", err)
	}
	defer bucket.Close()
	if err := downloadFile(ctx, bucket, "<key>", "<localFilePath>"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to download file: %v\n", err)
		os.Exit(1)
	}

}

func downloadFile(ctx context.Context, bucket *blob.Bucket, key, localFilePath string) error {
	// Create a reader for the object.
	reader, err := bucket.NewReader(ctx, key, nil)
	if err != nil {
		return fmt.Errorf("failed to create reader for %s: %v", key, err)
	}
	defer reader.Close()

	// Create a file to save the downloaded object.
	file, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", localFilePath, err)
	}
	defer file.Close()

	// Copy the object data to the local file.
	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to copy data to file %s: %v", localFilePath, err)
	}

	return nil
}
