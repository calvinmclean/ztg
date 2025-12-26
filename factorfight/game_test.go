package factorfight_test

import (
	"context"
	"testing"

	"ztg/dice"
	"ztg/factorfight"
)

func TestGame(t *testing.T) {
	const sides = 10
	dicePeer1, dicePeer2 := dice.NewChannelPeers()

	peer1, peer2 := factorfight.NewChannelPeers(dicePeer1, dicePeer2)

	p1, _ := factorfight.NewPlayer("P1", factorfight.DefaultStrategy, peer2)
	p2, _ := factorfight.NewPlayer("P2", factorfight.DefaultStrategy, peer1)

	ctx := context.Background()

	p1First := true
	for range 100 {
		errChan := make(chan error, 1)
		p1Chan := make(chan bool, 1)
		go func() {
			p1Win, err := p1.Play(ctx, p1First)
			p1Chan <- p1Win
			errChan <- err
		}()

		p2Win, err := p2.Play(ctx, !p1First)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = <-errChan
		if err != nil {
			t.Fatalf("unexpected error from P1: %v", err)
		}

		p1Win := <-p1Chan

		if p1Win == p2Win {
			t.Fatalf("unexpected win result: p1=%t, p2=%t", p1Win, p2Win)
		}

		p1First = !p1First
	}
}
