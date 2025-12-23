package dice_test

import (
	"context"
	"math"
	"net/http/httptest"
	"testing"

	"ztg/dice"
)

func TestPipePeer(t *testing.T) {
	const sides = 6

	peer1, peer2 := dice.NewPipePeers()

	d1, _ := dice.NewRoller("One", sides, peer1)
	d2, _ := dice.NewRoller("Two", sides, peer2)

	ctx := context.Background()
	roll1 := d1.Roll(ctx)
	roll2 := d2.Roll(ctx)

	r1, err := roll1.GetOne()
	if err != nil {
		t.Fatalf("unexpected error on Roll Result: %v", err)
	}

	r2, err := roll2.GetOne()
	if err != nil {
		t.Fatalf("unexpected error on Roll Result: %v", err)
	}

	if r1 != r2 {
		t.Errorf("rolls are not equal %d != %d", r1, r2)
	}
}

func TestDiceRollFairness(t *testing.T) {
	results := map[uint16]int{}

	const (
		rolls           = 10_000
		numDicePerRoll  = 8
		sides           = 10
		expectedPerSide = float64(rolls*numDicePerRoll) / sides
	)

	peer1, peer2 := dice.NewChannelPeers()
	d1, _ := dice.NewRoller("One", sides, peer1)
	d2, _ := dice.NewRoller("Two", sides, peer2)

	for range rolls {
		ctx := context.Background()
		roll1 := d1.Roll(ctx)
		roll2 := d2.Roll(ctx)

		r1, err := roll1.Get(numDicePerRoll)
		if err != nil {
			t.Fatalf("unexpected error on Roll Result: %v", err)
		}

		r2, err := roll2.Get(numDicePerRoll)
		if err != nil {
			t.Fatalf("unexpected error on Roll Result: %v", err)
		}

		for i := range numDicePerRoll {
			if r1[i] != r2[i] {
				t.Errorf("rolls are not equal %d != %d", r1, r2)
			}

			results[r1[i]]++
		}
	}

	sigma := math.Sqrt(expectedPerSide * (1 - 1.0/sides))
	min := expectedPerSide - 3*sigma
	max := expectedPerSide + 3*sigma

	t.Run("WithinRange", func(t *testing.T) {
		for i, c := range results {
			if float64(c) < min || float64(c) > max {
				t.Fatalf(
					"face %d outside 3σ range: got %d, expected %.1f ± %.1f. data: %v",
					i+1, c, expectedPerSide, 3*sigma, results,
				)
			}
		}
	})
}

func TestHTTP(t *testing.T) {
	peer1 := dice.NewHTTPPeer("")
	peer2 := dice.NewHTTPPeer("")

	server1 := httptest.NewServer(peer1)
	server2 := httptest.NewServer(peer2)

	peer1.SetSendAddr(server2.URL)
	peer2.SetSendAddr(server1.URL)

	const sides = 6
	d1, _ := dice.NewRoller("One", sides, peer1)
	d2, _ := dice.NewRoller("Two", sides, peer2)

	rolls := 100
	for range rolls {
		ctx := context.Background()
		roll1 := d1.Roll(ctx)
		roll2 := d2.Roll(ctx)

		r1, err := roll1.GetOne()
		if err != nil {
			t.Fatalf("unexpected error on Roll Result: %v", err)
		}

		r2, err := roll2.GetOne()
		if err != nil {
			t.Fatalf("unexpected error on Roll Result: %v", err)
		}

		if r1 != r2 {
			t.Errorf("rolls are not equal %d != %d", r1, r2)
		}
	}
}
