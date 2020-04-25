package deploy

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/savaki/fairy/internal/amazon/stack"
	"github.com/savaki/fairy/internal/banner"
)

// Upload contents of ${config.Dir}/resources to ${S3Bucket}/${S3Prefix}
func Upload(ctx context.Context, config Config) error {
	api := s3.New(config.Target)

	once := &sync.Once{}
	dir := filepath.Join(config.Dir, "resources")
	dir = strings.TrimRight(dir, "/") + "/"
	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		once.Do(func() {
			banner.Println("uploading resources ...")
		})

		rel := path
		if strings.HasPrefix(path, dir) {
			rel = rel[len(dir):]
		}

		bucket := config.Parameters[stack.S3Bucket]
		key := filepath.Join(config.Parameters[stack.S3Prefix], rel)

		defer func(begin time.Time) {
			log.Printf("uploaded %v -> s3://%v/%v (%v) - %v", path, bucket, key, time.Now().Sub(begin).Round(time.Millisecond), err)
		}(time.Now())

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to upload file, %v: %w", path, err)
		}
		defer f.Close()

		input := s3.PutObjectInput{
			Body:   f,
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}
		if _, err := api.PutObjectRequest(&input).Send(ctx); err != nil {
			return fmt.Errorf("failed to upload file, %v: %w", path, err)
		}

		return nil
	}
	if err := filepath.Walk(dir, fn); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("unable to read dir, %v: %w", dir, err)
	}

	return nil
}
