package minioClient

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"slices"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/hitalos/minioUp/config"
)

var client *minio.Client

func Init(cfg config.Config) error {
	var err error

	creds := credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")
	client, err = minio.New(cfg.Endpoint, &minio.Options{Secure: cfg.Secure, Creds: creds})

	return err
}

func UploadMultiple(ctx context.Context, dest config.Destination, filepaths []string, params []map[string]string) error {
	for idx, file := range filepaths {
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			return err
		}
		stat, _ := f.Stat()

		if err = Upload(ctx, dest, f, file, stat.Size(), params[idx]); err != nil {
			return err
		}
		_ = f.Close()
	}

	return nil
}

func Upload(ctx context.Context, dest config.Destination, r io.Reader, filename string, size int64, params map[string]string) error {
	originalFilename := filepath.Base(filename)

	if len(dest.AllowedTypes) > 0 {
		ext := filepath.Ext(originalFilename)[1:]
		if !slices.Contains(dest.AllowedTypes, ext) {
			return fmt.Errorf("invalid file type: %q", ext)
		}
	}

	for k, f := range dest.Fields {
		f.Value = params[k]
		if f.Validate() {
			continue
		}

		return fmt.Errorf("invalid value for field %q: %s", k, f.Value)
	}

	if size > dest.MaxUploadSize {
		return fmt.Errorf("file size exceeds the maximum allowed size of %d bytes", dest.MaxUploadSize)
	}

	options := minio.PutObjectOptions{
		UserMetadata: params,
		ContentType:  mime.TypeByExtension(filepath.Ext(filename)),
	}

	options.UserMetadata["originalFilename"] = originalFilename

	path := filepath.Join(dest.Prefix, originalFilename)
	if dest.Model != nil && dest.Model.Value != "" {
		path = filepath.Join(dest.Prefix, dest.MountName(options.UserMetadata))
	}

	_, err := client.PutObject(ctx, dest.Bucket, path, r, size, options)

	return err
}

func List(ctx context.Context, dest config.Destination) ([]minio.ObjectInfo, error) {
	if _, err := client.BucketExists(ctx, dest.Bucket); err != nil {
		return nil, err
	}

	opts := minio.ListObjectsOptions{Prefix: dest.Prefix, Recursive: true, WithMetadata: true}
	objCh := client.ListObjects(ctx, dest.Bucket, opts)
	list := make([]minio.ObjectInfo, 0)
	for obj := range objCh {
		list = append(list, obj)
	}

	return list, nil
}

func Delete(ctx context.Context, dest config.Destination, key string) error {
	return client.RemoveObject(
		ctx,
		dest.Bucket,
		filepath.Join(dest.Prefix, key),
		minio.RemoveObjectOptions{})
}
