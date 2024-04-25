FROM docker.io/library/golang:1.22-alpine as builder

WORKDIR /app
ADD go.mod go.sum ./
RUN go mod download

ADD . .
RUN CGO_ENABLED=0 go build -ldflags '-s -w' -o ./dist/minioUpServer ./cmd/server

FROM alpine

WORKDIR /app
COPY --from=builder /app/dist/minioUpServer ./

ENTRYPOINT ["/app/minioUpServer"]
