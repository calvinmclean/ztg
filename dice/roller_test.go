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

	d1 := dice.NewRoller("One", sides, peer1)
	d2 := dice.NewRoller("Two", sides, peer2)

	ctx := context.Background()
	roll1 := d1.Roll(ctx)
	roll2 := d2.Roll(ctx)

	r1, err := roll1.Result()
	if err != nil {
		t.Fatalf("unexpected error on Roll Result: %v", err)
	}

	r2, err := roll2.Result()
	if err != nil {
		t.Fatalf("unexpected error on Roll Result: %v", err)
	}

	if r1 != r2 {
		t.Errorf("rolls are not equal %d != %d", r1, r2)
	}
}

func TestDiceRollFairness(t *testing.T) {
	results := map[int]int{
		1: 0,
		2: 0,
		3: 0,
		4: 0,
		5: 0,
		6: 0,
	}

	const (
		rolls           = 10_000
		sides           = 6
		alpha           = 0.01 // chi-squared significance level
		expectedPerSide = float64(rolls) / sides
	)

	peer1, peer2 := dice.NewChannelPeers()
	d1 := dice.NewRoller("One", sides, peer1)
	d2 := dice.NewRoller("Two", sides, peer2)

	for range rolls {
		ctx := context.Background()
		roll1 := d1.Roll(ctx)
		roll2 := d2.Roll(ctx)

		r1, err := roll1.Result()
		if err != nil {
			t.Fatalf("unexpected error on Roll Result: %v", err)
		}

		r2, err := roll2.Result()
		if err != nil {
			t.Fatalf("unexpected error on Roll Result: %v", err)
		}

		if r1 != r2 {
			t.Errorf("rolls are not equal %d != %d", r1, r2)
		}

		results[r1]++
	}

	sigma := math.Sqrt(expectedPerSide * (1 - 1.0/sides))
	min := expectedPerSide - 3*sigma
	max := expectedPerSide + 3*sigma

	t.Run("WithinRange", func(t *testing.T) {
		for i, c := range results {
			if float64(c) < min || float64(c) > max {
				t.Fatalf(
					"face %d outside 3σ range: got %d, expected %.1f ± %.1f",
					i+1, c, expectedPerSide, 3*sigma,
				)
			}
		}
	})

	t.Run("ChiSquared", func(t *testing.T) {
		var chi2 float64
		for _, c := range results {
			diff := float64(c) - expectedPerSide
			chi2 += (diff * diff) / expectedPerSide
		}

		// Critical value for chi-squared with df = sides - 1
		// df = 5, alpha = 0.01 → 15.086
		const chi2Critical = 15.086

		if chi2 > chi2Critical {
			t.Fatalf(
				"chi-squared test failed: χ² = %.2f > %.3f (counts=%v)",
				chi2, chi2Critical, results,
			)
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
	d1 := dice.NewRoller("One", sides, peer1)
	d2 := dice.NewRoller("Two", sides, peer2)

	rolls := 100
	for range rolls {
		ctx := context.Background()
		roll1 := d1.Roll(ctx)
		roll2 := d2.Roll(ctx)

		r1, err := roll1.Result()
		if err != nil {
			t.Fatalf("unexpected error on Roll Result: %v", err)
		}

		r2, err := roll2.Result()
		if err != nil {
			t.Fatalf("unexpected error on Roll Result: %v", err)
		}

		if r1 != r2 {
			t.Errorf("rolls are not equal %d != %d", r1, r2)
		}
	}
}
