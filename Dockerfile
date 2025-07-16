FROM docker.io/library/golang:1.24-alpine as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-s -w' -o ./dist/minioUpServer ./cmd/server

FROM docker.io/library/alpine:3.22

WORKDIR /app
RUN apk --no-cache add tz libcrypto3=3.5.1-r0 libssl3=3.5.1-r0
COPY --from=builder /app/dist/minioUpServer ./

ENTRYPOINT ["/app/minioUpServer"]
