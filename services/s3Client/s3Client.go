package s3Client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/sync/errgroup"

	"github.com/hitalos/minioUp/config"
)

type (
	FileInfo struct {
		Name     string
		Size     int64
		LastMod  time.Time
		Metadata map[string]string
	}

	FileInfoList []FileInfo
)

func (il FileInfoList) Len() int           { return len(il) }
func (il FileInfoList) Swap(i, j int)      { il[i], il[j] = il[j], il[i] }
func (il FileInfoList) Less(i, j int) bool { return il[i].LastMod.Unix() > il[j].LastMod.Unix() }

var (
	client *s3.Client
	dec    = new(mime.WordDecoder)
	m      = new(sync.Mutex)
)

func Init(cfg config.Config) error {
	creds := credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")
	client = s3.NewFromConfig(aws.Config{
		Region:       cfg.Region,
		Credentials:  creds,
		AppID:        filepath.Base(os.Args[0]),
		BaseEndpoint: &cfg.Endpoint,
	}, func(o *s3.Options) {
		o.UsePathStyle = cfg.PathStyle
	})

	_, err := client.ListBuckets(context.Background(), &s3.ListBucketsInput{})

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

	mimetype := mime.TypeByExtension(filepath.Ext(filename))
	options := &s3.PutObjectInput{
		Bucket:      &dest.Bucket,
		Metadata:    params,
		ContentType: &mimetype,
		Body:        r,
	}

	options.Metadata["originalFilename"] = originalFilename

	path := filepath.Join(dest.Prefix, originalFilename)
	if dest.Model != nil && dest.Model.Value != "" {
		path = filepath.Join(dest.Prefix, dest.MountName(options.Metadata))
	}
	options.Key = &path

	_, err := client.PutObject(ctx, options)

	return err
}

func List(ctx context.Context, dest config.Destination) (FileInfoList, error) {
	opts := &s3.ListObjectsV2Input{Bucket: &dest.Bucket, Prefix: &dest.Prefix}
	list, err := client.ListObjectsV2(ctx, opts)
	if err != nil {
		return nil, err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(20)
	metadatas := map[string]map[string]string{}
	for _, obj := range list.Contents {
		o := obj
		g.Go(func() error {
			head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: &dest.Bucket,
				Key:    o.Key,
			})
			if err != nil {
				return nil
			}

			for k, v := range head.Metadata {
				if strings.HasPrefix(v, "=?UTF-8?") {
					s, err := dec.Decode(v)
					if err != nil {
						slog.Error(err.Error())
						continue
					}
					head.Metadata[k] = string(s)
				}
			}

			m.Lock()
			metadatas[*obj.Key] = head.Metadata
			m.Unlock()

			return nil
		})
	}
	_ = g.Wait()

	prefixLen := len(dest.Prefix)
	if prefixLen > 0 {
		prefixLen++
	}

	fInfoList := make(FileInfoList, 0)
	for _, obj := range list.Contents {
		fInfoList = append(fInfoList, FileInfo{
			(*obj.Key)[prefixLen:],
			*obj.Size,
			obj.LastModified.Local(),
			metadatas[*obj.Key],
		})
	}

	return fInfoList, nil

}

func Delete(ctx context.Context, dest config.Destination, key string) error {
	path := filepath.Join(dest.Prefix, key)
	opts := &s3.DeleteObjectInput{
		Bucket: &dest.Bucket,
		Key:    &path,
	}

	_, err := client.DeleteObject(ctx, opts)

	return err
}
