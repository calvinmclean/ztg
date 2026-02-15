package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/calvinmclean/ztg"
	"github.com/calvinmclean/ztg/factorfight"
	ffserver "github.com/calvinmclean/ztg/server/factorfight"

	"github.com/gregdel/pushover"
)

func main() {
	// Create context that listens for interrupt signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// Optional Pushover notifications
	var pushoverClient *notifyClient
	pushoverAppToken := os.Getenv("PUSHOVER_APP_TOKEN")
	pushoverRecipientToken := os.Getenv("PUSHOVER_RECIPIENT_TOKEN")
	if pushoverAppToken != "" && pushoverRecipientToken != "" {
		var err error
		pushoverClient, err = newNotifyClient(pushoverAppToken, pushoverRecipientToken)
		if err != nil {
			return fmt.Errorf("error creating Pushover client: %w", err)
		}
	}

	cfg, err := ztg.LoadConfig()
	if err != nil {
		return err
	}

	ffCfg := ffserver.Config{
		Strategy: factorfight.DefaultStrategy,
		OnGameComplete: func(win bool, log factorfight.GameLog) {
			winText := "Lose!"
			if win {
				winText = "Win!"
			}
			fmt.Printf("FactorFight Game Result: %s\n", winText)
			fmt.Println(log)

			if pushoverClient == nil {
				return
			}

			// Send Pushover notification
			err := pushoverClient.send("FactorFight Game", fmt.Sprintf("Result: %s", winText))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to send Pushover notification: %v\n", err)
			}
		},
	}

	return ztg.Run(
		ctx, cfg,
		ztg.WithFactorFight(ffCfg),
		ztg.WithHighRoll(),
	)
}

type notifyClient struct {
	app       *pushover.Pushover
	recipient *pushover.Recipient
}

func newNotifyClient(appToken, recipientToken string) (*notifyClient, error) {
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
