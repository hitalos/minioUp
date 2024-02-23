package minioClient

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/hitalos/minioUp/config"
)

func Upload(cfg config.Config, filepaths []string, params [][]string) error {
	client, err := New(cfg)
	if err != nil {
		return err
	}

	for idx, file := range filepaths {
		originalFilename := filepath.Base(file)

		options := minio.PutObjectOptions{
			UserMetadata: map[string]string{"originalFilename": originalFilename},
		}
		if strings.Join(params[idx], "|") != "" {
			options.UserMetadata["params"] = strings.Join(params[idx], "|")
		}

		tmpl := cfg.Dest.Template
		bucket := cfg.Dest.Bucket
		filename := mountName(tmpl, append([]string{originalFilename}, params[idx]...))
		path := filepath.Join(cfg.Dest.Prefix, filename)
		_, err = client.FPutObject(context.Background(), bucket, path, file, options)
		if err != nil {
			return err
		}
	}

	return nil
}

func New(cfg config.Config) (*minio.Client, error) {
	creds := credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")

	return minio.New(cfg.Endpoint, &minio.Options{Secure: cfg.Secure, Creds: creds})
}

func mountName(templateString string, params []string) string {
	if templateString == "" {
		return filepath.Base(params[0])
	}

	tmpl, err := template.New("").Funcs(sprig.FuncMap()).Parse(templateString)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	str := new(bytes.Buffer)
	if err := tmpl.Execute(str, params); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return str.String()
}

func List(cfg config.Config) error {
	client, err := New(cfg)
	if err != nil {
		return err
	}

	if _, err := client.BucketExists(context.Background(), cfg.Dest.Bucket); err != nil {
		return err
	}

	opts := minio.ListObjectsOptions{Prefix: cfg.Dest.Prefix, Recursive: true}
	objCh := client.ListObjects(context.Background(), cfg.Dest.Bucket, opts)
	for obj := range objCh {
		fmt.Println(obj.Key[len(cfg.Dest.Prefix)+1:])
	}

	return nil
}
