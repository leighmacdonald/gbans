package asset

//
// import (
//	"context"
//	"errors"
//	"fmt"
//	"io"
//	"log/slog"
//	"sync"
//
//	sq "github.com/Masterminds/squirrel"
//	"github.com/gofrs/uuid/v5"
//	"github.com/leighmacdonald/gbans/internal/database"
//	"github.com/leighmacdonald/gbans/internal/domain"
//	"github.com/minio/minio-go/v7"
// )
//
// type s3repository struct {
//	mu     sync.RWMutex
//	client *minio.Client
//	region string
//	db     database.Database
// }
//
// func NewS3Repository(db database.Database, client *minio.Client, region string) domain.AssetRepository {
//	return &s3repository{
//		mu:     sync.RWMutex{},
//		client: client,
//		db:     db,
//		region: region,
//	}
// }
//
// func (r *s3repository) Init(ctx context.Context) error {
//	if errDemo := r.CreateBucketIfNotExists(ctx, "demo"); errDemo != nil {
//		return errDemo
//	}
//
//	if errMedia := r.CreateBucketIfNotExists(ctx, "media"); errMedia != nil {
//		return errMedia
//	}
//
//	return nil
// }
//
// func (r *s3repository) Get(ctx context.Context, bucket string, name string) (io.Reader, error) {
//	reader, err := r.client.GetObject(ctx, bucket, name, minio.GetObjectOptions{})
//	if err != nil {
//		return nil, errors.Join(err, domain.ErrAssetGet)
//	}
//
//	return reader, nil
// }
//
// func (r *s3repository) GetAsset(ctx context.Context, uuid uuid.UUID) (domain.Asset, error) {
//	var asset domain.Asset
//
//	row, errRow := r.db.QueryRowBuilder(ctx, r.db.Builder().
//		Select("asset_id", "bucket", "path", "name", "mime_type", "size", "old_id").
//		From("asset").Where(sq.Eq{"asset_id": uuid.String()}))
//
//	if errRow != nil {
//		return asset, errRow
//	}
//
//	if errScan := row.Scan(&asset.AssetID, &asset.Bucket, &asset.Path, &asset.Name, &asset.MimeType, &asset.Size, &asset.OldID); errScan != nil {
//		return asset, r.db.DBErr(errScan)
//	}
//
//	return asset, nil
// }
//
// func (r *s3repository) CreateBucketIfNotExists(ctx context.Context, name string) error {
//	r.mu.Lock()
//	defer r.mu.Unlock()
//
//	exists, errBucketExists := r.client.BucketExists(ctx, name)
//	if errBucketExists != nil && !exists {
//		return errors.Join(errBucketExists, domain.ErrBucketCheck)
//	}
//
//	if !exists {
//		errMake := r.client.MakeBucket(ctx, name, minio.MakeBucketOptions{Region: r.region})
//		if errMake != nil {
//			return errors.Join(errMake, domain.ErrBucketCreate)
//		}
//
//		slog.Info("Created new S3Store bucket", slog.String("name", name))
//	}
//
//	// string ???
//	policy := fmt.Sprintf(`{
//		"Version": "2012-10-17",
//		"Statement": [
//			{
//				"Sid": "PublicReadGetObject",
//				"Effect": "Allow",
//				"Principal": "*",
//				"Action": [
//					"s3:GetObject"
//				],
//				"Resource": [
//					"arn:aws:s3:::%s/*"
//				]
//			}
//		]
//		}`, name)
//
//	if err := r.client.SetBucketPolicy(ctx, name, policy); err != nil {
//		return errors.Join(err, domain.ErrPolicy)
//	}
//
//	return nil
// }
//
// func (r *s3repository) Put(ctx context.Context, bucket string, name string, body io.Reader, size int64, contentType string) error {
//	r.mu.Lock()
//	defer r.mu.Unlock()
//
//	_, err := r.client.PutObject(ctx, bucket, name, body, size, minio.PutObjectOptions{
//		ContentType: contentType,
//	})
//	if err != nil {
//		return errors.Join(err, domain.ErrWriteObject)
//	}
//
//	slog.Debug("File uploaded successfully",
//		slog.String("name", name),
//		slog.String("bucket", bucket))
//
//	return nil
// }
//
// func (r *s3repository) Remove(ctx context.Context, bucket string, name string) error {
//	r.mu.Lock()
//	defer r.mu.Unlock()
//
//	if err := r.client.RemoveObject(ctx, bucket, name, minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
//		return errors.Join(err, domain.ErrDeleteObject)
//	}
//
//	return nil
// }
//
// func (r *s3repository) LinkObject(bucket string, name string) string {
//	endpoint := r.client.EndpointURL()
//	endpoint.Path = bucket + "/" + name
//
//	return endpoint.String()
// }
//
// func (r *s3repository) SaveAsset(ctx context.Context, asset *domain.Asset) error {
//	return r.db.DBErr(r.db.ExecInsertBuilder(ctx, r.db.
//		Builder().
//		Insert("asset").
//		Columns("asset_id", "bucket", "path", "name", "mime_type", "size", "old_id").
//		Values(asset.AssetID, asset.Bucket, asset.Path, asset.Name, asset.MimeType, asset.Size, asset.OldID)))
// }
//
// func (r *s3repository) DeleteAsset(ctx context.Context, asset *domain.Asset) error {
//	return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, r.db.
//		Builder().
//		Delete("asset").
//		Where(sq.Eq{"asset_id": asset.AssetID})))
// }
