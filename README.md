# Minio Uploader

Web interface to upload files to AWS S3 or [Minio](https://min.io/) (AWS S3 compatible self-hosted service).

You can try this project with a local instance of [Minio](https://min.io/). There is a task on `Makefile` (minio-server) to run a local container. You will need [docker](https://www.docker.com) or [podman](https://podman.io) to run it. After start your minio container, access it on `http://localhost:9000` and create your buckets.

## Installing

```shell
go install github.com/hitalos/minioUp/cmd/minioUp@latest
```

## Configuration

Create a `config.yml` file in the current directory. Use this [`config.example.yml`](config.example.yml) as a reference.
The `params` will be used to rename the uploaded files using [golang template](https://golang.org/pkg/text/template/) syntax with [sprig](https://masterminds.github.io/sprig/) package functions.

## Run

```shell
minioUp <path-to-file> "<params>"
```

Important: the "params" is mandatory, even if empty. Why the command wait for pairs.

After uploading, the command will list the files in the destination for you to check.

Use:

```shell
minioUp -l
```

To only list the files in the destination.

P.S.: If you running `minioUp` without a "standard input" (ex.: inside a crontab script), it will always choose the first destination, without asking.

## Examples

Considering the following configuration:

```yaml
port: ":8000"
endpoint: minio.domain.com
secure: false
accessKey: minio
secretKey: minio
allowedHosts:
  - localhost:8000

destinations:
  - bucket: uploads
    prefix: "2024/03" # optional
    name: "uploads - march" # will be showed as "uploads - march" on menu
    allowedTypes: ["jpg", "png", "pdf"]
    template: # optional
      model: "{{ lower (index . 0) }}"
      pattern: "regex pattern" # optional
      example: "placeholder text" # optional
      description: "label description text" # optional
    webhook: # optional
      url: https://yourwebhookurl.com/api/webhook
      method: POST # optional
      headers: # optional
        key1: value1
        key2: value2
      fields: # optional
        key1: value1
        key2: value2
```

With the following command:

```shell
minioUp FILE.PDF ""
```

The file will be uploaded to `uploads` bucket with the original name (`FILE.PDF`) and a prefix `2024/03`.

If you define the destination `template` to `tmp_{{ lower (index . 0) }}`, the file will be renamed to `tmp_file.pdf`.
