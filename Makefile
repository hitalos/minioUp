all: dist/minioUp

dist/minioUp:
	CGO_ENABLED=0 go build -ldflags '-s -w' -o ./dist/minioUp ./cmd/minioUp

install:
	go install ./cmd/minioUp

clean:
	rm -rf ./dist

CONTAINER_RUNTINE=$(shell [ -e /usr/bin/podman ] && echo podman || echo docker)
minio-server:
	@echo "Creating minio volumes if not exists…"
	$(CONTAINER_RUNTINE) volume create --ignore minio-data
	@echo "Creating a ssh key on volume…"
	$(eval DATADIR=$(shell $(CONTAINER_RUNTINE) volume inspect minio-data | grep Mountpoint | sed -e 's/^.*: //g' -e 's/[\b",]//g'))
	ssh-keygen -f $(DATADIR)/id_rsa -N '' -t rsa
	$(CONTAINER_RUNTINE) run -d \
		--name minio \
		-v minio-data:/data \
		-p 9000:9000 \
		-p 9001:9001 \
		-p 9022:9022 \
		-e MINIO_ROOT_USER=minio \
		-e MINIO_ROOT_PASSWORD='minio-root-password' \
		docker.io/minio/minio server /data --console-address ":9001" --sftp="address=:9022" --sftp="ssh-private-key=/data/id_rsa"

.PHONY: all clean dist/minioUp install minio-server