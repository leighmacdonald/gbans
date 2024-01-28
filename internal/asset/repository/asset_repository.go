package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

type s3repository struct {
	mu     sync.RWMutex
	client *minio.Client
	log    *zap.Logger
	region string
	db     database.Database
}

func NewS3Repository(logger *zap.Logger, db database.Database, client *minio.Client, region string) domain.AssetRepository {
	// TODO init client outside
	//config := cu.Config()
	//// Initialize minio client object.
	//minioClient, err := minio.New(config.S3.Endpoint, &minio.Options{
	//	Creds:  credentials.NewStaticV4(config.S3.AccessKey, config.S3.SecretKey, ""),
	//	Secure: config.S3.SSL,
	//})
	//if err != nil {
	//	return nil, errors.Join(err, domain.ErrInitClient)
	//}

	return &s3repository{
		mu:     sync.RWMutex{},
		client: client,
		db:     db,
		log:    logger.Named("s3"),
		region: region,
	}
}

func (r *s3repository) CreateBucketIfNotExists(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	errMake := r.client.MakeBucket(ctx, name, minio.MakeBucketOptions{Region: r.region})
	if errMake != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := r.client.BucketExists(ctx, name)
		if errBucketExists != nil && !exists {
			return errors.Join(errBucketExists, domain.ErrBucketCheck)
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

	if err := r.client.SetBucketPolicy(ctx, name, policy); err != nil {
		return errors.Join(err, domain.ErrPolicy)
	}

	r.log.Info("Successfully created new bucket", zap.String("name", name))

	return nil
}

func (r *s3repository) Put(ctx context.Context, bucket string, name string, body io.Reader, size int64, contentType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.client.PutObject(ctx, bucket, name, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return errors.Join(err, domain.ErrWriteObject)
	}

	r.log.Debug("File uploaded successfully",
		zap.String("name", name),
		zap.String("bucket", bucket))

	return nil
}

func (r *s3repository) Remove(ctx context.Context, bucket string, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.client.RemoveObject(ctx, bucket, name, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
		return errors.Join(err, domain.ErrDeleteObject)
	}

	return nil
}

func (r *s3repository) LinkObject(bucket string, name string) string {
	endpoint := r.client.EndpointURL()
	endpoint.Path = bucket + "/" + name

	return endpoint.String()
}

func (r *s3repository) SaveAsset(ctx context.Context, asset *domain.Asset) error {
	return r.db.DBErr(r.db.ExecInsertBuilder(ctx, r.db.
		Builder().
		Insert("asset").
		Columns("asset_id", "bucket", "path", "name", "mime_type", "size", "old_id").
		Values(asset.AssetID, asset.Bucket, asset.Path, asset.Name, asset.MimeType, asset.Size, asset.OldID)))
}

func (r *s3repository) DeleteAsset(ctx context.Context, asset *domain.Asset) error {
	return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, r.db.
		Builder().
		Delete("asset").
		Where(sq.Eq{"asset_id": asset.AssetID})))
}
