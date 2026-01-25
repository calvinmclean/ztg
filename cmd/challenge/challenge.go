package challenge

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/calvinmclean/ztg/config"
	gamepb "github.com/calvinmclean/ztg/gen/go/game/v1"
	"github.com/calvinmclean/ztg/grpcutil"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/server"

	"github.com/urfave/cli/v3"
)

var Command = &cli.Command{
	Name:  "challenge",
	Usage: "Create, sign, and send a challenge message",
	Commands: []*cli.Command{
		{
			Name:  "send",
			Usage: "Send a signed challenge to a server",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "target",
					Aliases:  []string{"t"},
					Required: true,
					Usage:    "Target server address (e.g., localhost:50052)",
				},
				&cli.StringFlag{
					Name:     "game",
					Aliases:  []string{"g"},
					Required: true,
					Usage:    "Game ID (e.g., ztg.FactorFight.v1, ztg.HighRoll.v1)",
				},
				&cli.StringFlag{
					Name:     "server",
					Aliases:  []string{"s"},
					Required: true,
					Usage:    "This server address (for signing)",
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
			Action: sendChallenge,
		},
		{
			Name:  "create",
			Usage: "Create a signed challenge message (print to stdout)",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "target",
					Aliases:  []string{"t"},
					Required: true,
					Usage:    "Target server address (e.g., localhost:50052)",
				},
				&cli.StringFlag{
					Name:     "game",
					Aliases:  []string{"g"},
					Required: true,
					Usage:    "Game ID (e.g., ztg.FactorFight.v1, ztg.HighRoll.v1)",
				},
				&cli.StringFlag{
					Name:     "server",
					Aliases:  []string{"s"},
					Required: true,
					Usage:    "This server address (for signing)",
				},
				&cli.StringFlag{
					Name:     "key-path",
					Aliases:  []string{"k"},
					Required: false,
					Usage:    "Private key path (defaults to keys/example_ed25519.pem)",
					Value:    "keys/example_ed25519.pem",
				},
			},
			Action: createChallenge,
		},
	},
}

func sendChallenge(ctx context.Context, cmd *cli.Command) error {
	target := cmd.String("target")
	gameID := cmd.String("game")
	serverAddr := cmd.String("server")
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

	// Create challenge request
	req := &gamepb.ChallengeRequest{
		Target: target,
		GameId: gameID,
	}

	signature, err := signer.SignProto(req)
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	signedReq := &gamepb.SignedChallengeRequest{
		Challenge: req,
		Signature: signature,
	}

	// Connect to target server
	conn, err := grpcutil.NewClient(serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to target server: %w", err)
	}
	defer conn.Close()

	// Create client and send request
	client := gamepb.NewGameServiceClient(conn)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := client.Challenge(ctx, signedReq)
	if err != nil {
		return fmt.Errorf("challenge request failed: %w", err)
	}

	fmt.Printf("Challenge sent successfully!\n")
	fmt.Printf("Target: %s\n", target)
	fmt.Printf("Game: %s\n", gameID)
	if resp.Win != nil {
		fmt.Printf("Win: %v\n", *resp.Win)
	}
	if resp.Message != "" {
		fmt.Printf("Message: %s\n", resp.Message)
	}

	return nil
}

func createChallenge(ctx context.Context, cmd *cli.Command) error {
	target := cmd.String("target")
	gameID := cmd.String("game")
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

	// Create challenge request
	req := &gamepb.ChallengeRequest{
		Target: target,
		GameId: gameID,
	}

	signature, err := signer.SignProto(req)
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	signedReq := &gamepb.SignedChallengeRequest{
		Challenge: req,
		Signature: signature,
	}

	// Output the signed request
	fmt.Printf("SignedChallengeRequest created:\n")
	fmt.Printf("  Target: %s\n", signedReq.Challenge.Target)
	fmt.Printf("  GameID: %s\n", signedReq.Challenge.GameId)
	fmt.Printf("  Signature.SignerAddress: %s\n", signedReq.Signature.SignerAddress)
	fmt.Printf("  Signature.Signature: %s\n", hex.EncodeToString(signedReq.Signature.Signature))

	return nil
}
