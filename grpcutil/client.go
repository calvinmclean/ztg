package grpcutil

import (
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClient creates a gRPC client connection with appropriate TLS or insecure credentials
// based on the server address. If the address ends with :443, it uses TLS credentials,
// otherwise it uses insecure credentials.
func NewClient(address string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if strings.HasSuffix(address, ":443") {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return grpc.NewClient(address, opts...)
}
