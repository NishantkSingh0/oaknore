package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	appconfig "github.com/oaknore/pms3/internal/config"
)

type S3Service struct {
	client        *s3.Client
	uploader      *manager.Uploader
	presignClient *s3.PresignClient
	bucket        string
	baseURL       string
	presignExpiry time.Duration
}

func NewS3Service(cfg appconfig.AWSConfig) (*S3Service, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	return &S3Service{
		client:        client,
		uploader:      manager.NewUploader(client),
		presignClient: s3.NewPresignClient(client),
		bucket:        cfg.S3Bucket,
		baseURL:       cfg.S3BaseURL,
		presignExpiry: cfg.PresignExpiry,
	}, nil
}

// UploadFile streams a multipart file to S3 under the given prefix.
// Returns the S3 key and public URL.
func (s *S3Service) UploadFile(ctx context.Context, prefix string, fh *multipart.FileHeader) (key, url string, err error) {
	f, err := fh.Open()
	if err != nil {
		return "", "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	ext := filepath.Ext(fh.Filename)
	key = fmt.Sprintf("%s/%s%s", strings.Trim(prefix, "/"), uuid.New().String(), ext)

	_, err = s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        f,
		ContentType: aws.String(fh.Header.Get("Content-Type")),
	})
	if err != nil {
		return "", "", fmt.Errorf("s3 upload: %w", err)
	}

	url = fmt.Sprintf("%s/%s", strings.TrimRight(s.baseURL, "/"), key)
	return key, url, nil
}

// PresignGet returns a time-limited presigned GET URL for a private object.
func (s *S3Service) PresignGet(ctx context.Context, key string) (string, error) {
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(s.presignExpiry))
	if err != nil {
		return "", fmt.Errorf("presign: %w", err)
	}
	return req.URL, nil
}

// DeleteFile removes an object from S3.
func (s *S3Service) DeleteFile(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}
