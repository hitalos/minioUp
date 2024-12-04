FROM docker.io/library/golang:1.23-alpine as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-s -w' -o ./dist/minioUpServer ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /app/dist/minioUpServer ./

ENTRYPOINT ["/app/minioUpServer"]
