package dice_test

import (
	"math"
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

	for range rolls {
		d1 := dice.NewRoller("One")
		d2 := dice.NewRoller("Two")

		roll1 := d1.Roll(d2)
		roll2 := d2.Roll(d1)

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
