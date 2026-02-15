package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/calvinmclean/ztg/config"
	"github.com/calvinmclean/ztg/factorfight"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/server"
	ffserver "github.com/calvinmclean/ztg/server/factorfight"
	highrollserver "github.com/calvinmclean/ztg/server/highroll"

	"github.com/gregdel/pushover"
)

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.Config{}
	config.LoadFromEnv(&cfg)
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	pushoverAppToken := os.Getenv("PUSHOVER_APP_TOKEN")
	pushoverRecipientToken := os.Getenv("PUSHOVER_RECIPIENT_TOKEN")
	if pushoverAppToken == "" || pushoverRecipientToken == "" {
		return errors.New("missing PUSHOVER_APP_TOKEN and/or PUSHOVER_RECIPIENT_TOKEN")
	}

	pushoverClient, err := newNotifyClient(pushoverAppToken, pushoverRecipientToken)
	if err != nil {
		return fmt.Errorf("error creating Pushover client: %w", err)
	}

	keyManager, err := identity.NewKeyManager(cfg.Identity)
	if err != nil {
		return fmt.Errorf("failed to initialize key manager: %w", err)
	}

	grpcServer, err := server.NewServer(cfg.Server, cfg.Database, keyManager)
	if err != nil {
		return fmt.Errorf("server initialization failed: %w", err)
	}

	ffCfg := ffserver.Config{
		Strategy: factorfight.DefaultStrategy,
		OnGameComplete: func(win bool, _ factorfight.GameLog) {
			winText := "Lose!"
			if win {
				winText = "Win!"
			}
			pushoverClient.send("FactorFight Game", fmt.Sprintf("Result: %s", winText))
		},
	}
	sqlStore := grpcServer.GetStore()

	ffService := ffserver.NewService(ffCfg, keyManager, cfg.Identity.ServerAddress, sqlStore, grpcServer.GetLogger())
	grpcServer.Register(ffService)

	highrollService := highrollserver.NewService(keyManager, cfg.Identity.ServerAddress, sqlStore)
	grpcServer.Register(highrollService)

	return grpcServer.Run()
}

type notifyClient struct {
	app       *pushover.Pushover
	recipient *pushover.Recipient
}

func newNotifyClient(appToken, recipientToken string) (*notifyClient, error) {
	if appToken == "" {
		return nil, errors.New("missing required app_token")
	}
	if recipientToken == "" {
		return nil, errors.New("missing required recipient_token")
	}

	return &notifyClient{
		app:       pushover.New(appToken),
		recipient: pushover.NewRecipient(recipientToken),
	}, nil
}

func (c *notifyClient) send(title, message string) error {
	msg := pushover.NewMessageWithTitle(message, title)
	_, err := c.app.SendMessage(msg, c.recipient)
	return err
}
