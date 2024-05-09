package minioClient

import (
	"context"
	"io"
	"mime"
	"os"
	"path/filepath"

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

func UploadMultiple(dest config.Destination, filepaths []string, params []map[string]string) error {
	for idx, file := range filepaths {
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			return err
		}
		stat, _ := f.Stat()

		if err = Upload(dest, f, file, stat.Size(), params[idx]); err != nil {
			return err
		}
		_ = f.Close()
	}

	return nil
}

func Upload(dest config.Destination, r io.Reader, filename string, size int64, params map[string]string) error {
	originalFilename := filepath.Base(filename)

	options := minio.PutObjectOptions{
		UserMetadata: params,
		ContentType:  mime.TypeByExtension(filepath.Ext(filename)),
	}

	options.UserMetadata["originalFilename"] = originalFilename

	path := filepath.Join(dest.Prefix, originalFilename)
	if dest.Model != "" {
		path = filepath.Join(dest.Prefix, dest.MountName(options.UserMetadata))
	}

	_, err := client.PutObject(context.Background(), dest.Bucket, path, r, size, options)

	return err
}

func List(dest config.Destination) ([]minio.ObjectInfo, error) {
	if _, err := client.BucketExists(context.Background(), dest.Bucket); err != nil {
		return nil, err
	}

	opts := minio.ListObjectsOptions{Prefix: dest.Prefix, Recursive: true, WithMetadata: true}
	objCh := client.ListObjects(context.Background(), dest.Bucket, opts)
	list := make([]minio.ObjectInfo, 0)
	for obj := range objCh {
		list = append(list, obj)
	}

	return list, nil
}

func Delete(dest config.Destination, key string) error {
	return client.RemoveObject(context.Background(), dest.Bucket, filepath.Join(dest.Prefix, key), minio.RemoveObjectOptions{})
}
