FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ztg ./cmd/ztg

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
RUN addgroup -g 1001 -S ztg && \
    adduser -u 1001 -S ztg -G ztg
WORKDIR /app
COPY --from=builder /app/ztg /usr/local/bin/ztg
RUN chown -R ztg:ztg /usr/local/bin/ztg
USER ztg
ENTRYPOINT ["ztg"]
CMD ["--help"]
