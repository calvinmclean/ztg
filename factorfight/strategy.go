package factorfight

import "context"

var DefaultStrategy = ClosestMoveStrategy{}

// Strategy allows defining an external implementation that chooses the preferred move
type Strategy interface {
	// ChooseMove receives a list of all possible moves and returns the one that it wants to execute
	ChooseMove(ctx context.Context, state State, moves []Move) Move
}

// ClosestMoveStrategy always picks the move closest to the goal
type ClosestMoveStrategy struct{}

var _ Strategy = ClosestMoveStrategy{}

// ChooseMove chooses the move closest to the goal
func (ClosestMoveStrategy) ChooseMove(_ context.Context, state State, moves []Move) Move {
	best := moves[0]
	for _, m := range moves {
		// Skip all Pawn2 moves for now since it is not used
		if m.Pawn2.Result != 0 {
			continue
		}
		if m.Pawn1.Result > best.Pawn1.Result {
			best = m
		}
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
