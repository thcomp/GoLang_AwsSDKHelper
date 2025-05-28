package awssdkhelper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	ThcompUtility "github.com/thcomp/GoLang_Utility"
)

type S3Helper struct {
	config      *aws.Config
	credentials *aws.Credentials
	bucket      string
	client      *s3.Client
	logger      *ThcompUtility.Logger

	createdByFunc bool
}

func CreateS3Helper(accessKeyId, secretAccessKey, region, bucket string, logger *ThcompUtility.Logger) *S3Helper {
	ret := &S3Helper{
		config: &aws.Config{
			Region: region,
		},
		credentials: &aws.Credentials{
			AccessKeyID:     accessKeyId,
			SecretAccessKey: secretAccessKey,
		},
		bucket:        bucket,
		logger:        logger,
		createdByFunc: true,
	}

	ret.client = s3.NewFromConfig(*ret.config)

	return ret
}

func (s3Helper *S3Helper) ListItems(prefix string, continuationToken *string) (items [](*S3Item), nextContinuationToken *string, err error) {
	ctx := context.Background()
	output, listErr := s3Helper.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:            &s3Helper.bucket,
		ContinuationToken: continuationToken,
		Delimiter:         aws.String("/"),
		Prefix:            &prefix,
	})

	if listErr == nil {
		nextContinuationToken = output.NextContinuationToken

		if len(output.Contents) > 0 {
			items = [](*S3Item){}
			for _, content := range output.Contents {
				if content.Key != nil && (*content.Key) != prefix {
					items = append(
						items,
						&S3Item{
							IsDir:        false,
							Path:         (*content.Key),
							size:         content.Size,
							lastModified: content.LastModified,
							helper:       s3Helper,
						},
					)
				}
			}
		}

		if len(output.CommonPrefixes) > 0 {
			for _, commonPrefix := range output.CommonPrefixes {
				if commonPrefix.Prefix != nil {
					items = append(
						items,
						&S3Item{
							IsDir:  true,
							Path:   (*commonPrefix.Prefix),
							helper: s3Helper,
						},
					)
				}
			}
		}
	} else {
		err = listErr
	}

	return
}

func (s3Helper *S3Helper) PutItem(item *S3Item) (err error) {
	ctx := context.Background()

	mimeType := ThcompUtility.GetMIMETypeFromExtension(item.Path)
	_, err = s3Helper.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s3Helper.bucket,
		Key:         &item.Path,
		Body:        item, // You need to provide a valid io.Reader here
		ContentType: aws.String(mimeType),
	})

	return
}

func (s3Helper *S3Helper) PutData(itemKey string, data []byte) (err error) {
	ctx := context.Background()
	mimeType := ThcompUtility.GetMIMETypeFromExtension(itemKey)

	reader := bytes.NewReader(data)
	_, err = s3Helper.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s3Helper.bucket,
		Key:         &itemKey,
		Body:        reader, // You need to provide a valid io.Reader here
		ContentType: aws.String(mimeType),
	})

	return
}

func (s3Helper *S3Helper) PutFile(itemKey string, filepath string) (err error) {
	ctx := context.Background()

	if reader, readErr := os.Open(filepath); readErr == nil {
		defer reader.Close()

		mimeType := ThcompUtility.GetMIMETypeFromExtension(filepath)
		_, err = s3Helper.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      &s3Helper.bucket,
			Key:         &itemKey,
			Body:        reader, // You need to provide a valid io.Reader here
			ContentType: aws.String(mimeType),
		})
	}

	return
}

func (s3Helper *S3Helper) DeleteItem(itemKey string) (err error) {
	ctx := context.Background()

	_, err = s3Helper.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s3Helper.bucket,
		Key:    &itemKey,
	})

	return
}

type S3Item struct {
	Path         string
	IsDir        bool
	lastModified *time.Time
	size         *int64
	helper       *S3Helper
	reader       io.ReadCloser
}

func (item *S3Item) Reader() (reader io.ReadCloser, retErr error) {
	if item.reader == nil {
		ctx := context.Background()
		if output, err := item.helper.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &item.helper.bucket,
			Key:    &item.Path,
		}); err == nil {
			item.reader = output.Body
			reader = item.reader
		} else {
			retErr = err
		}
	}

	return
}

func (item *S3Item) Read(buffer []byte) (size int, retErr error) {
	if reader, err := item.Reader(); err == nil {
		size, retErr = reader.Read(buffer)
	} else {
		retErr = err
	}

	return
}

func (item *S3Item) Close() (retErr error) {
	if item.reader != nil {
		if err := item.reader.Close(); err != nil {
			retErr = err
		}
		item.reader = nil
	}

	return
}

func (item *S3Item) Size() (int64, error) {
	if item.size != nil {
		return *item.size, nil
	}

	return -1, fmt.Errorf("size is nil for item %s", item.Path)
}

func (item *S3Item) LastModified() (time.Time, error) {
	if item.lastModified != nil {
		return *item.lastModified, nil
	}

	return time.Time{}, fmt.Errorf("lastModified is nil for item %s", item.Path)
}

func (item *S3Item) LastModifiedNano() (int64, error) {
	if item.lastModified != nil {
		return item.lastModified.UnixNano(), nil
	}

	return 0, fmt.Errorf("lastModified is nil for item %s", item.Path)
}
