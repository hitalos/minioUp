FROM docker.io/library/golang:1.26.1-alpine as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-s -w' -o ./dist/minioUpServer ./cmd/server

FROM docker.io/library/alpine:3.23

WORKDIR /app
RUN apk --no-cache add tz=0.8.0-r10
COPY --from=builder /app/dist/minioUpServer ./

ENTRYPOINT ["/app/minioUpServer"]
