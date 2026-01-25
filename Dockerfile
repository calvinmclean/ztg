# WARNING: this Dockerfile is for example/server, not for the main program. This is required because
# the example server uses a local import instead of github.com/calvinmclean/ztg
#
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/example/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ztg .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
RUN addgroup -g 1001 -S ztg && \
    adduser -u 1001 -S ztg -G ztg
WORKDIR /app
COPY --from=builder /app/example/server/ztg /usr/local/bin/ztg
RUN chown -R ztg:ztg /usr/local/bin/ztg
USER ztg
ENTRYPOINT ["ztg"]
