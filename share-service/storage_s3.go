//go:build s3

package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// s3Store persists shares in an S3 bucket, one object per share id. This is
// the recommended backend for AWS deployments (cheap, durable, no servers).
type s3Store struct {
	bucket string
	client *s3.Client
}

func makeS3Store() (store, error) {
	bucket := os.Getenv("QSIM_SHARES_BUCKET")
	if bucket == "" {
		return nil, &missingBucketError{}
	}
	return &s3Store{bucket: bucket}, nil
}

type missingBucketError struct{}

func (e *missingBucketError) Error() string {
	return "STORAGE=s3 requires QSIM_SHARES_BUCKET"
}

func (s *s3Store) open() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	s.client = s3.NewFromConfig(cfg)
	if _, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)}); err != nil {
		log.Printf("s3: bucket %s not reachable (will retry on writes): %v", s.bucket, err)
	}
	return nil
}

func (s *s3Store) save(id string, data string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(id),
		Body:        strings.NewReader(data),
		ContentType: aws.String("text/plain"),
	})
	return err
}

func (s *s3Store) load(id string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),
	})
	if err != nil {
		return "", err
	}
	defer out.Body.Close()
	buf := make([]byte, 0, 64*1024)
	tmp := make([]byte, 32*1024)
	for {
		n, err := out.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return string(buf), nil
}