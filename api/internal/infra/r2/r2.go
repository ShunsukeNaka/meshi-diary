// Package r2 は Cloudflare R2 を S3 互換 API で操作する。
package r2

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/config"
)

type Client struct {
	s3      *s3.Client
	presign *s3.PresignClient
	bucket  string
}

// New は R2 用の S3 クライアントを作る。ここではネットワークにはつながない。
func New(ctx context.Context, cfg config.R2Config) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
		// R2 は S3 の新しいチェックサムに対応していないものがあるので、必要なときだけ付ける。
		// 署名付き URL に余計なチェックサムのパラメータが入るのも防ぐ。
		awsconfig.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		awsconfig.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID))
	})

	return &Client{
		s3:      client,
		presign: s3.NewPresignClient(client),
		bucket:  cfg.Bucket,
	}, nil
}

// Put はオブジェクトを保存する。
func (c *Client) Put(ctx context.Context, key, contentType string, body io.Reader) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

// Get はオブジェクトを読み込む。
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read object %q: %w", key, err)
	}
	return data, nil
}

// PresignGet は表示用の期限付き URL を発行する。
func (c *Client) PresignGet(ctx context.Context, key string, expires time.Duration) (string, error) {
	req, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", fmt.Errorf("presign get %q: %w", key, err)
	}
	return req.URL, nil
}

// Delete はオブジェクトを削除する。存在しないキーでもエラーにならない（S3 の仕様）。
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	return nil
}
