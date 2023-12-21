package app

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type AssetStore interface {
	Put(ctx context.Context, bucket string, name string, body io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, bucket string, name string) error
}

type S3Client struct {
	*sync.RWMutex
	*minio.Client
	log    *zap.Logger
	region string
	ssl    bool
}

func NewS3Client(log *zap.Logger, endpoint string, accessKey string, secretKey string, useSSL bool, region string) (*S3Client, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize minio client")
	}

	return &S3Client{Client: minioClient, log: log, region: region, ssl: useSSL, RWMutex: &sync.RWMutex{}}, nil
}

func (s3 *S3Client) CreateBucketIfNotExists(ctx context.Context, name string) error {
	s3.Lock()
	defer s3.Unlock()

	errMake := s3.MakeBucket(ctx, name, minio.MakeBucketOptions{Region: s3.region})
	if errMake != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := s3.BucketExists(ctx, name)
		if errBucketExists != nil && !exists {
			return errors.Wrap(errBucketExists, "Failed to check if bucket exists")
		}
	}

	// string ???
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "PublicReadGetObject",
				"Effect": "Allow",
				"Principal": "*",
				"Action": [
					"s3:GetObject"
				],
				"Resource": [
					"arn:aws:s3:::%s/*"
				]
			}
		]
		}`, name)

	if err := s3.SetBucketPolicy(ctx, name, policy); err != nil {
		return errors.Wrap(err, "Failed to set bucket policy")
	}

	s3.log.Info("Successfully created new bucket", zap.String("name", name))

	return nil
}

func (s3 *S3Client) Put(ctx context.Context, bucket string, name string, body io.Reader, size int64, contentType string) error {
	s3.Lock()
	defer s3.Unlock()

	_, err := s3.PutObject(ctx, bucket, name, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to write object to s3 store")
	}

	s3.log.Debug("File uploaded successfully",
		zap.String("name", name),
		zap.String("bucket", bucket))

	return nil
}

func (s3 *S3Client) Remove(ctx context.Context, bucket string, name string) error {
	s3.Lock()
	defer s3.Unlock()

	if err := s3.RemoveObject(ctx, bucket, name, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
		return errors.Wrap(err, "Failed to delete object")
	}

	s3.log.Debug("File deleted successfully",
		zap.String("name", name),
		zap.String("bucket", bucket))

	return nil
}

func (s3 *S3Client) LinkObject(bucket string, name string) string {
	endpoint := s3.EndpointURL()
	endpoint.Path = bucket + "/" + name

	return endpoint.String()
}

func GenerateFileMeta(body io.Reader, name string) (string, string, int64, error) {
	content, errRead := io.ReadAll(body)
	if errRead != nil {
		return "", "", 0, errors.Wrap(errRead, "Failed to read file content")
	}

	mime := mimetype.Detect(content)

	if !strings.HasSuffix(strings.ToLower(name), mime.Extension()) {
		name += mime.Extension()
	}

	return name, mime.String(), int64(len(content)), nil
}
