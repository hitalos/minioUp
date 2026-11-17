all: dist/minioUp dist/minioUpServer

dist/minioUp:
	CGO_ENABLED=0 go build -ldflags '-s -w' -trimpath -o ./dist/minioUp ./cmd/minioUp

dist/minioUpServer:
	CGO_ENABLED=0 go build -ldflags '-s -w' -trimpath -o ./dist/minioUpServer ./cmd/server

dev:
	ENV=dev go run -tags dev -ldflags "-X github.com/hitalos/minioUp/cmd/server/i18n.defaultLocale=$(LANG)" ./cmd/server

install:
	go install ./cmd/minioUp

clean:
	rm -rf ./dist

CONTAINER_RUNTIME=$(shell [ -e /usr/bin/podman ] && echo podman || echo docker)
minio-server:
	@echo "Creating volumes if not exists…"
	$(CONTAINER_RUNTIME) volume create --ignore minio-data
	@echo "Creating a ssh key on volume…"
	$(eval DATADIR=$(shell $(CONTAINER_RUNTIME) volume inspect minio-data | grep Mountpoint | sed -e 's/^.*: //g' -e 's/[\b",]//g'))
	ssh-keygen -f $(DATADIR)/id_rsa -N '' -t rsa
	$(CONTAINER_RUNTIME) run -d \
		--name minio \
		-v minio-data:/data \
		-p 9000:9000 \
		-p 9001:9001 \
		-p 9022:9022 \
		-e MINIO_ROOT_USER=minio \
		-e MINIO_ROOT_PASSWORD='minio-root-password' \
		docker.io/minio/minio server /data --console-address ":9001" --sftp="address=:9022" --sftp="ssh-private-key=/data/id_rsa"

lint:
	go fix --diff ./...
	golangci-lint run ./...

sec:
	govulncheck ./...
	gosec ./...
	trivy fs .
	grype . --add-cpes-if-none

CONTAINER_RUNTIME=$(shell [ -e /usr/bin/buildah ] && echo buildah || echo docker)
container-image:
	$(CONTAINER_RUNTIME) build -t docker.io/$(USER)/minioup:latest .

container-image-push:
	$(CONTAINER_RUNTIME) push docker.io/$(USER)/minioup:latest

container-image-sec:
	trivy image docker.io/$(USER)/minioup:latest
	grype docker.io/$(USER)/minioup:latest

.PHONY: all clean container-image container-image-sec dist/minioUp dist/minioUpServer install lint sec
