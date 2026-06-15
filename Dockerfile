FROM docker.io/library/golang:1.26.4-alpine as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-s -w' -o ./dist/minioUpServer ./cmd/server

FROM docker.io/library/alpine:3.24

WORKDIR /app
RUN apk --no-cache add tz=0.8.0-r13
COPY --from=builder /app/dist/minioUpServer ./

ENTRYPOINT ["/app/minioUpServer"]
