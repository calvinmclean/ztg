package factorfight

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"ztg/dice"
)

// TODO: Add 2nd pawn
// TODO: Handle "bump"
// TODO: logging for play-by-play (also record moves)
// TODO: initialize game between players (who goes first?)
// TODO: make sure all validMoves are calculated. Should this be part of Strategy or built-in to the game?

const goal = 101

type Peer interface {
	SendTurn(context.Context, Turn) error
	RecvTurn(context.Context) (Turn, error)
	Dice() dice.Peer
}

type Turn struct {
	Pawn1Pos int
	Pawn2Pos int
}

// State represents the current state of the game
type State struct {
	Pawn1Pos     int
	Pawn2Pos     int
	PeerPawn1Pos int
	PeerPawn2Pos int
}

// Turn updates the pawn positions according to the turn
func (s *State) Turn(t Turn) {
	s.Pawn1Pos = t.Pawn1Pos
	s.Pawn2Pos = t.Pawn2Pos
}

// Turn updates the peer's pawn positions according to the turn
func (s *State) PeerTurn(t Turn) {
	s.PeerPawn1Pos = t.Pawn1Pos
	s.PeerPawn2Pos = t.Pawn2Pos
}

type Player struct {
	name     string
	roller   dice.Roller
	peer     Peer
	strategy Strategy
}

func NewPlayer(name string, strategy Strategy, peer Peer) (Player, error) {
	roller, err := dice.NewRoller(name, 10, peer.Dice())
	if err != nil {
		return Player{}, fmt.Errorf("error creating roller: %w", err)
	}

	if strategy == nil {
		strategy = DefaultStrategy
	}

	return Player{
		name:     name,
		roller:   roller,
		peer:     peer,
		strategy: strategy,
	}, nil
}

func (p *Player) Play(ctx context.Context, goFirst bool) (bool, error) {
	state := &State{}

	if !goFirst {
		err := p.OtherTurn(ctx, state)
		if err != nil {
			return false, fmt.Errorf("error doing other player's first turn: %w", err)
		}
	}

	turnNum := 0
	for {
		// fmt.Printf("%s (%d): Taking turn\n", p.name, turnNum)
		err := p.TakeTurn(ctx, state)
		if err != nil {
			return false, fmt.Errorf("error taking turn: %w", err)
		}

		// fmt.Printf("%s (%d): Moving %d -> %d\n", p.name, turnNum, pos, newPos)
		if state.Pawn1Pos == goal {
			return true, nil
		}

		err = p.OtherTurn(ctx, state)
		if err != nil {
			return false, fmt.Errorf("error doing other player's first turn: %w", err)
		}

		if state.PeerPawn1Pos == goal {
			return false, nil
		}

		turnNum++
	}
}

func (p *Player) OtherTurn(ctx context.Context, state *State) error {
	rolls, err := p.roller.RollSync(ctx, 2)
	if err != nil {
		return fmt.Errorf("error rolling: %w", err)
	}

	peerTurn, err := p.peer.RecvTurn(ctx)
	if err != nil {
		return fmt.Errorf("error receiving turn: %w", err)
	}

	err = validateTurn(rolls, *state, peerTurn)
	if err != nil {
		return err
	}

	state.PeerTurn(peerTurn)

	return nil
}

func (p *Player) TakeTurn(ctx context.Context, state *State) error {
	rolls, err := p.roller.RollSync(ctx, 2)
	if err != nil {
		return fmt.Errorf("error rolling: %w", err)
	}

	d1, d2 := int(rolls[0]), int(rolls[1])

	// fmt.Printf("%s (%d): Roll [%d, %d]\n", p.name, i, d1, d2)

	moves := validMoves(state.Pawn1Pos, d1, d2)
	if len(moves) == 0 {
		return errors.New("no valid moves")
	}

	turn := p.strategy.ChooseMove(ctx, *state, moves)
	state.Turn(turn)

	err = p.peer.SendTurn(ctx, turn)
	if err != nil {
		return fmt.Errorf("error sending turn: %w", err)
	}

	return nil
}

func validMoves(pos, d1, d2 int) []int {
	moves := []int{}

	// Addition
	if pos+d1+d2 <= goal {
		moves = append(moves, pos+d1+d2)
	}

	// Subtraction
	if pos-(d1+d2) >= 0 {
		moves = append(moves, pos-(d1+d2))
	}

	// Multiplication
	if pos == 0 {
		if d1*d2 <= goal {
			moves = append(moves, d1*d2)
		}
	} else if pos*(d1*d2) <= goal {
		moves = append(moves, pos*(d1*d2))
	}

	// Division (exact only)
	product := d1 * d2
	if product != 0 && pos%product == 0 {
		result := pos / product
		if result >= 0 {
			moves = append(moves, result)
		}
	}

	return moves
}

func validateTurn(rolls []uint16, state State, turn Turn) error {
	d1, d2 := int(rolls[0]), int(rolls[1])

	moves := validMoves(state.PeerPawn1Pos, d1, d2)
	if len(moves) == 0 {
		return errors.New("no valid moves")
	}

	if slices.Contains(moves, turn.Pawn1Pos) {
		return nil
	}
	return fmt.Errorf("other player completed invalid move: [%d, %d] %d -> %d", d1, d2, state.PeerPawn1Pos, state.Pawn1Pos)
}
