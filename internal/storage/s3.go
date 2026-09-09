package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/masoniclounge/masoniccore/internal/config"
)

// S3 wraps the MinIO client and the default bucket used for media uploads.
type S3 struct {
	client *minio.Client
	bucket string
}

// New creates an S3 client, verifies connectivity and ensures the bucket exists.
func New(ctx context.Context, cfg config.Config) (*S3, error) {
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.S3Bucket)
	if err != nil {
		return nil, fmt.Errorf("check s3 bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.S3Bucket, minio.MakeBucketOptions{Region: cfg.S3Region}); err != nil {
			return nil, fmt.Errorf("create s3 bucket: %w", err)
		}
	}
	return &S3{client: client, bucket: cfg.S3Bucket}, nil
}

// Bucket returns the default bucket name.
func (s *S3) Bucket() string {
	return s.bucket
}

// Upload stores an object under key in the default bucket.
func (s *S3) Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("upload object %s: %w", key, err)
	}
	return nil
}

// Open streams an object from the default bucket.
func (s *S3) Open(ctx context.Context, key string) (*minio.Object, error) {
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

// Remove deletes an object from the default bucket. Missing objects are ignored.
func (s *S3) Remove(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object %s: %w", key, err)
	}
	return nil
}
