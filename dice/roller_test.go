package dice_test

import (
	"context"
	"io"
	"math"
	"net/http/httptest"
	"testing"

	"ztg/dice"
)

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

	rd1, wr1 := io.Pipe()
	rd2, wr2 := io.Pipe()
	tport1 := dice.NewRWTransport(rd1, wr2)
	tport2 := dice.NewRWTransport(rd2, wr1)

	d1 := dice.NewRoller("One", sides, tport1)
	d2 := dice.NewRoller("Two", sides, tport2)

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
	ta := dice.NewHTTPTransport("")
	tb := dice.NewHTTPTransport("")

	taServer := httptest.NewServer(ta)
	tbServer := httptest.NewServer(tb)

	ta.SetSendAddr(tbServer.URL)
	tb.SetSendAddr(taServer.URL)

	const sides = 6
	d1 := dice.NewRoller("One", sides, ta)
	d2 := dice.NewRoller("Two", sides, tb)

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
