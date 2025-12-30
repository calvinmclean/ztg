package factorfight

import "testing"

func TestBumpDetection(t *testing.T) {
	// Generate moves for Pawn1 starting at 48 and Pawn2 starting at 51
	moves := generateMoves(48, 51, 2, 3, 50, 0)

	for _, move := range moves {
		if move.Pawn1.Result == 50 && move.Pawn1.Bump != BumpedPeer1 {
			t.Fatalf("Bump not detected for Pawn1 at position 50")
		}
		if move.Pawn2.Result == 50 && move.Pawn2.Bump != BumpedPeer1 {
			t.Fatalf("Bump not detected for Pawn2 at position 50")
		}
	}
}
