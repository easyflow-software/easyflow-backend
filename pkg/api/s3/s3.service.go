package s3

import (
	"context"
	"easyflow-backend/pkg/api"
	"easyflow-backend/pkg/common"
	"easyflow-backend/pkg/enum"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
)

/*
Private function to connect to the S3 bucket
*/
func connect(cfg *common.Config) (*s3.Client, error) {
	config, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.BucketAccessKeyId, cfg.BucketSecret, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(config, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.BucketURL)
	})

	return client, nil
}

/*
Object upload url generation
*/
// TODO: Add filetype restriction to the upload url
func GenerateUploadURL(logger *common.Logger, cfg *common.Config, bucketName string, objectKey string, expiration int) (*string, *api.ApiError) {
	client, err := connect(cfg)
	if err != nil {
		logger.PrintfError("An error happened while connecting to the bucket %s", bucketName)
		return nil, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	presigner := s3.NewPresignClient(client)

	req, err := presigner.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &objectKey,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expiration) * time.Second
	})
	if err != nil {
		logger.PrintfError("Could not get object %s in bucket %s", objectKey, bucketName)
		return nil, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	return &req.URL, nil
}

/*
GetObjectsWithPrefix returns a list of objects with a given prefix in the bucket
*/
func GetObjectsWithPrefix(logger *common.Logger, cfg *common.Config, bucketName string, prefix string) (*s3.ListObjectsV2Output, *api.ApiError) {
	client, err := connect(cfg)
	if err != nil {
		logger.PrintfError("An error happened while connecting to the bucket %s", bucketName)
		return nil, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	listedObjects, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &bucketName,
		Prefix: &prefix,
	})
	if err != nil {
		logger.PrintfError("An error happened while listing objects in bucket %s", bucketName)
		return nil, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	return listedObjects, nil
}

/*
GenerateDownloadURL returns a presigned URL for an object in the bucket
*/
func GenerateDownloadURL(logger *common.Logger, cfg *common.Config, bucketName string, objectKey string, expiration int) (*string, *api.ApiError) {
	client, err := connect(cfg)
	if err != nil {
		logger.PrintfError("An error happened while connecting to the bucket %s", bucketName)
		return nil, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	exists, err := client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: &bucketName,
		Key:    &objectKey,
	})
	if err != nil || exists == nil {
		logger.PrintfWarning("Could not get object %s in bucket %s", objectKey, bucketName)
		return nil, &api.ApiError{
			Code:    http.StatusNoContent,
			Error:   enum.NotFound,
			Details: err,
		}
	}

	presigner := s3.NewPresignClient(client)

	req, err := presigner.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &objectKey,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expiration) * time.Second
	})
	if err != nil {
		logger.PrintfError("Could not presign url to get object %s in bucket %s", objectKey, bucketName)
		return nil, &api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	return &req.URL, nil
}
