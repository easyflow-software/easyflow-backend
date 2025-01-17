package minio

import (
	"easyflow-backend/pkg/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// connect initializes a MinIO client
func connect(cfg *config.Config) (*minio.Client, error) {
	client, err := minio.New(cfg.BucketURL, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.BucketAccessKeyId, cfg.BucketSecret, ""),
		Secure: true,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}
