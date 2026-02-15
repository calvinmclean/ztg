package identity

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"
	"github.com/calvinmclean/ztg/grpcutil"

	"github.com/urfave/cli/v3"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var Command = &cli.Command{
	Name:  "identity",
	Usage: "Manage identities in the store",
	Commands: []*cli.Command{
		{
			Name:  "add",
			Usage: "Add an identity to the store",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "server",
					Aliases:  []string{"s"},
					Required: true,
					Usage:    "Target server address",
				},
				&cli.StringFlag{
					Name:     "peer",
					Required: true,
					Usage:    "Peer server address to add",
				},
				&cli.StringFlag{
					Name:     "peer-name",
					Required: true,
					Usage:    "Peer server name",
				},
				&cli.StringFlag{
					Name:     "owner-name",
					Required: true,
					Usage:    "Peer owner name",
				},
				&cli.StringFlag{
					Name:     "public-key",
					Required: true,
					Usage:    "Peer public key in hex or base64 format",
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
			Action: addIdentity,
		},
		{
			Name:  "list",
			Usage: "List all identities in the store",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "server",
					Aliases:  []string{"s"},
					Required: true,
					Usage:    "Target server address",
				},
				&cli.IntFlag{
					Name:     "timeout",
					Aliases:  []string{"T"},
					Required: false,
					Usage:    "Request timeout in seconds",
					Value:    30,
				},
			},
			Action: listIdentities,
		},
	},
}

func addIdentity(ctx context.Context, cmd *cli.Command) error {
	serverAddr := cmd.String("server")
	peerAddr := cmd.String("peer")
	peerName := cmd.String("peer-name")
	ownerName := cmd.String("owner-name")
	publicKeyHex := cmd.String("public-key")
	timeout := time.Duration(cmd.Int("timeout")) * time.Second

	// Parse public key (try hex first, then base64)
	var publicKey []byte
	var err error

	// Try hex decoding first
	if strings.HasPrefix(strings.ToLower(publicKeyHex), "0x") {
		publicKeyHex = publicKeyHex[2:] // Remove 0x prefix
	}

	publicKey, err = hex.DecodeString(publicKeyHex)
	if err != nil {
		// If hex fails, try base64
		publicKey, err = base64.StdEncoding.DecodeString(publicKeyHex)
		if err != nil {
			return fmt.Errorf("failed to decode public key as hex or base64: %w", err)
		}
	}

	// Create identity message
	identity := &identitypb.Identity{
		PublicKey:     publicKey,
		ServerAddress: peerAddr,
		ServerName:    peerName,
		OwnerName:     ownerName,
		Capabilities:  []string{"ztg.FactorFight.v1", "ztg.HighRoll.v1"},
		CreatedAt:     timestamppb.Now(),
		IsTrusted:     false,
	}

	// Create request
	req := &identitypb.AddIdentityRequest{
		Identity: identity,
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

	_, err = client.AddIdentity(ctx, req)
	if err != nil {
		return fmt.Errorf("add identity request failed: %w", err)
	}

	fmt.Printf("Identity added successfully!\n")
	fmt.Printf("Target: %s\n", serverAddr)
	fmt.Printf("Peer: %s (%s)\n", peerAddr, peerName)
	fmt.Printf("Owner: %s\n", ownerName)

	return nil
}

func listIdentities(ctx context.Context, cmd *cli.Command) error {
	serverAddr := cmd.String("server")
	timeout := time.Duration(cmd.Int("timeout")) * time.Second

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

	resp, err := client.ListIdentities(ctx, &identitypb.ListIdentitiesRequest{})
	if err != nil {
		return fmt.Errorf("list identities request failed: %w", err)
	}

	if len(resp.Identities) == 0 {
		fmt.Printf("No identities found in store at %s\n", serverAddr)
		return nil
	}

	fmt.Printf("Identities in store at %s:\n", serverAddr)
	fmt.Printf("%-20s %-15s %-15s %-10s %-20s\n", "Server Address", "Server Name", "Owner Name", "Trusted", "Public Key")
	fmt.Printf("%s\n", fmt.Sprintf("%-20s %-15s %-15s %-10s %-20s", "----------------", "----", "----", "----", "----"))

	for _, identity := range resp.Identities {
		trusted := "No"
		if identity.IsTrusted {
			trusted = "Yes"
		}
		publicKeyHex := hex.EncodeToString(identity.PublicKey)
		if len(publicKeyHex) > 20 {
			publicKeyHex = publicKeyHex[:17] + "..."
		}

		fmt.Printf("%-20s %-15s %-15s %-10s %-20s\n",
			identity.ServerAddress,
			identity.ServerName,
			identity.OwnerName,
			trusted,
			publicKeyHex,
		)
	}

	return nil
}
