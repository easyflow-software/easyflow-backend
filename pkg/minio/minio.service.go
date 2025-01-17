package minio

import (
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/logger"

	"context"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
)

// GenerateUploadURL creates a presigned URL for uploading an object
func GenerateUploadURL(logger *logger.Logger, cfg *config.Config, bucketName, objectKey string, expiration int) (*string, *errors.ApiError) {
	client, err := connect(cfg)
	if err != nil {
		logger.PrintfError("Error connecting to bucket %s: %v", bucketName, err)
		return nil, &errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	presignedURL, err := client.PresignedPutObject(context.Background(), bucketName, objectKey, time.Duration(expiration)*time.Second)
	if err != nil {
		logger.PrintfError("Error generating presigned upload URL for object %s in bucket %s: %v", objectKey, bucketName, err)
		return nil, &errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	urlStr := presignedURL.String()
	return &urlStr, nil
}

// GetObjectsWithPrefix lists objects in a bucket with the specified prefix
func GetObjectsWithPrefix(logger *logger.Logger, cfg *config.Config, bucketName, prefix string) ([]minio.ObjectInfo, *errors.ApiError) {
	client, err := connect(cfg)
	if err != nil {
		logger.PrintfError("Error connecting to bucket %s: %v", bucketName, err)
		return nil, &errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	objectCh := client.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var objects []minio.ObjectInfo
	for object := range objectCh {
		if object.Err != nil {
			logger.PrintfError("Error listing object with prefix %s in bucket %s: %v", prefix, bucketName, object.Err)
			return nil, &errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   enum.ApiError,
				Details: object.Err,
			}
		}
		objects = append(objects, object)
	}

	return objects, nil
}

// GenerateDownloadURL creates a presigned URL for downloading an object
func GenerateDownloadURL(logger *logger.Logger, cfg *config.Config, bucketName, objectKey string, expiration int) (*string, *errors.ApiError) {
	client, err := connect(cfg)
	if err != nil {
		logger.PrintfError("Error connecting to bucket %s: %v", bucketName, err)
		return nil, &errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	// Check if the object exists
	_, err = client.StatObject(context.Background(), bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		logger.PrintfWarning("Object %s not found in bucket %s: %v", objectKey, bucketName, err)
		return nil, &errors.ApiError{
			Code:    http.StatusNotFound,
			Error:   enum.NotFound,
			Details: err,
		}
	}

	presignedURL, err := client.PresignedGetObject(context.Background(), bucketName, objectKey, time.Duration(expiration)*time.Second, nil)
	if err != nil {
		logger.PrintfError("Error generating presigned download URL for object %s in bucket %s: %v", objectKey, bucketName, err)
		return nil, &errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		}
	}

	urlStr := presignedURL.String()
	return &urlStr, nil
}
