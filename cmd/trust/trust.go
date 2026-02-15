package trust

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/calvinmclean/ztg/config"
	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"
	"github.com/calvinmclean/ztg/grpcutil"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/server"

	"github.com/urfave/cli/v3"
)

var Command = &cli.Command{
	Name:  "trust",
	Usage: "Manage trust status of identities",
	Commands: []*cli.Command{
		{
			Name:  "send",
			Usage: "Send a request to set trust status of an identity",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "trusted",
					Aliases:  []string{"r"},
					Required: true,
					Usage:    "Trust status (true or false)",
				},
				&cli.StringFlag{
					Name:     "server",
					Aliases:  []string{"s"},
					Required: true,
					Usage:    "This server address",
				},
				&cli.StringFlag{
					Name:     "peer",
					Required: true,
					Usage:    "The address of the trusted peer",
				},
				&cli.StringFlag{
					Name:     "key-path",
					Aliases:  []string{"k"},
					Required: false,
					Usage:    "Private key path (defaults to keys/example_ed25519.pem)",
					Value:    "keys/example_ed25519.pem",
				},
				&cli.IntFlag{
					Name:     "timeout",
					Aliases:  []string{"T"},
					Required: false,
					Usage:    "Request timeout in seconds",
					Value:    30,
				},
			},
			Action: sendTrustRequest,
		},
		{
			Name:  "create",
			Usage: "Create a signed trust request message (print to stdout)",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "trusted",
					Aliases:  []string{"r"},
					Required: true,
					Usage:    "Trust status (true or false)",
				},
				&cli.StringFlag{
					Name:     "server",
					Aliases:  []string{"s"},
					Required: true,
					Usage:    "This server address",
				},
				&cli.StringFlag{
					Name:     "peer",
					Required: true,
					Usage:    "The address of the trusted peer",
				},
				&cli.StringFlag{
					Name:     "key-path",
					Aliases:  []string{"k"},
					Required: false,
					Usage:    "Private key path (defaults to keys/example_ed25519.pem)",
					Value:    "keys/example_ed25519.pem",
				},
			},
			Action: createTrustRequest,
		},
	},
}

func sendTrustRequest(ctx context.Context, cmd *cli.Command) error {
	serverAddr := cmd.String("server")
	peer := cmd.String("peer")
	trusted := cmd.Bool("trusted")
	keyPath := cmd.String("key-path")
	timeout := time.Duration(cmd.Int("timeout")) * time.Second

	// Create key manager
	identityConfig := config.IdentityConfig{
		ServerAddress:  "owner",
		PrivateKeyFile: keyPath,
	}

	km, err := identity.NewKeyManager(identityConfig)
	if err != nil {
		return fmt.Errorf("failed to create key manager: %w", err)
	}

	// Create signer
	signer := server.NewSigner(km, "owner")

	// Create trust request message
	message := fmt.Appendf(nil, "set-trust:%s:%v", peer, trusted)

	// Sign the message
	signature, err := signer.SignMessage(message)
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	// Create request
	req := &identitypb.SetTrustRequest{
		ServerAddress: peer,
		Trusted:       trusted,
		Signature:     signature,
	}

	// Connect to target server
	conn, err := grpcutil.NewClient(serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to target server: %w", err)
	}
	defer conn.Close()

	// Create client and send request
	client := identitypb.NewIdentityServiceClient(conn)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err = client.SetTrust(ctx, req)
	if err != nil {
		return fmt.Errorf("set trust request failed: %w", err)
	}

	fmt.Printf("Trust status updated successfully!\n")
	fmt.Printf("Target: %s\n", serverAddr)
	fmt.Printf("Trusted: %v\n", trusted)

	return nil
}

func createTrustRequest(ctx context.Context, cmd *cli.Command) error {
	trusted := cmd.Bool("trusted")
	peer := cmd.String("peer")
	serverAddr := cmd.String("server")
	keyPath := cmd.String("key-path")

	// Create key manager
	identityConfig := config.IdentityConfig{
		ServerAddress:      serverAddr,
		PrivateKeyFile:     keyPath,
		OwnerPublicKeyFile: "keys/example_ed25519.pub.pem",
	}

	km, err := identity.NewKeyManager(identityConfig)
	if err != nil {
		return fmt.Errorf("failed to create key manager: %w", err)
	}

	// Create signer
	signer := server.NewSigner(km, serverAddr)

	// Create trust request message
	message := fmt.Appendf(nil, "set-trust:%s:%v", serverAddr, trusted)

	// Sign the message
	signature, err := signer.SignMessage(message)
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	// Create request
	req := &identitypb.SetTrustRequest{
		ServerAddress: peer,
		Trusted:       trusted,
		Signature:     signature,
	}

	// Output the signed request
	fmt.Printf("SetTrustRequest created:\n")
	fmt.Printf("  Trusted: %v\n", req.Trusted)
	fmt.Printf("  Signature.SignerAddress: %s\n", req.Signature.SignerAddress)
	fmt.Printf("  Signature.Signature: %s\n", hex.EncodeToString(req.Signature.Signature))

	return nil
}
