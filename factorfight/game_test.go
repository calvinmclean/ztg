package factorfight_test

import (
	"context"
	"testing"

	"ztg/dice"
	"ztg/factorfight"
)

func TestBumps(t *testing.T) {
	state := &factorfight.State{
		Pawn1Pos:     5,
		Pawn2Pos:     10,
		PeerPawn1Pos: 8,
		PeerPawn2Pos: 12,
	}

	move := factorfight.Move{
		Pawn1: factorfight.PawnMove{Result: 12, Bump: factorfight.BumpedPeer2},
		Pawn2: factorfight.PawnMove{Result: 20},
	}
	state.Move(move)
	if state.Pawn1Pos == 0 {
		t.Errorf("expected no bump")
	}
	if state.PeerPawn2Pos != 0 {
		t.Errorf("expected PeerPawn2 to reset on bump, got %d", state.PeerPawn2Pos)
	}

	state = &factorfight.State{
		Pawn1Pos: 5,
		Pawn2Pos: 10,
	}
	move = factorfight.Move{
		Pawn1: factorfight.PawnMove{Result: 10},
		Pawn2: factorfight.PawnMove{Result: 10, Bump: factorfight.BumpedSelf},
	}
	state.Move(move)
	if state.Pawn1Pos != 0 {
		t.Errorf("expected Pawn1 to reset on self bump, got %d", state.Pawn1Pos)
	}
}

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
			p1Win, _, err := p1.Play(ctx, p1First)
			p1Chan <- p1Win
			errChan <- err
		}()

		p2Win, _, err := p2.Play(ctx, !p1First)
		if err != nil {
			if gameErr, ok := err.(*factorfight.GameError); ok {
				t.Fatalf("unexpected error: %v\nGame log:\n%s", err, gameErr.GameLog.String())
			}
			t.Fatalf("unexpected error: %v", err)
		}

		err = <-errChan
		if err != nil {
			if gameErr, ok := err.(*factorfight.GameError); ok {
				t.Fatalf("unexpected error from P1: %v\nGame log:\n%s", err, gameErr.GameLog.String())
			}
			t.Fatalf("unexpected error from P1: %v", err)
		}

		p1Win := <-p1Chan

		if p1Win == p2Win {
			t.Fatalf("unexpected win result: p1=%t, p2=%t", p1Win, p2Win)
		}

		p1First = !p1First
	}
}
