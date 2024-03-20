package minioClient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
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

func UploadMultiple(dest config.Destination, filepaths []string, params [][]string) error {
	for idx, file := range filepaths {
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			return err
		}

		if err = Upload(dest, f, file, params[idx]); err != nil {
			return err
		}
		_ = f.Close()
	}

	return nil
}

func Upload(dest config.Destination, r io.Reader, filename string, params []string) error {
	originalFilename := filepath.Base(filename)

	options := minio.PutObjectOptions{
		UserMetadata: map[string]string{"originalFilename": originalFilename},
		ContentType:  mime.TypeByExtension(filepath.Ext(filename)),
	}

	if strings.Join(params, "|") != "" {
		options.UserMetadata["params"] = strings.Join(params, "|")
	}

	path := filepath.Join(dest.Prefix, originalFilename)
	if dest.Template != nil {
		tmpl := dest.Template.Model
		path = filepath.Join(dest.Prefix, mountName(tmpl, append([]string{originalFilename}, params...)))
	}

	_, err := client.PutObject(context.Background(), dest.Bucket, path, r, -1, options)

	return err
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

func List(dest config.Destination) ([]minio.ObjectInfo, error) {
	if _, err := client.BucketExists(context.Background(), dest.Bucket); err != nil {
		return nil, err
	}

	opts := minio.ListObjectsOptions{Prefix: dest.Prefix, Recursive: true}
	objCh := client.ListObjects(context.Background(), dest.Bucket, opts)
	list := make([]minio.ObjectInfo, 0)
	for obj := range objCh {
		list = append(list, obj)
	}

	return list, nil
}
