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
	SendMove(context.Context, Move) error
	RecvMove(context.Context) (Move, error)
	Dice() dice.Peer
}

// State represents the current state of the game
type State struct {
	Pawn1Pos     int
	Pawn2Pos     int
	PeerPawn1Pos int
	PeerPawn2Pos int
}

// Move updates the pawn positions according to the Move
func (s *State) Move(m Move) {
	s.Pawn1Pos = m.Pawn1.Result
	s.Pawn2Pos = m.Pawn2.Result
}

// PeerMove updates the peer's pawn positions according to the Move
func (s *State) PeerMove(m Move) {
	s.PeerPawn1Pos = m.Pawn1.Result
	s.PeerPawn2Pos = m.Pawn2.Result
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

	peerMove, err := p.peer.RecvMove(ctx)
	if err != nil {
		return fmt.Errorf("error receiving turn: %w", err)
	}

	err = validateMove(rolls, *state, peerMove)
	if err != nil {
		return err
	}

	state.PeerMove(peerMove)

	return nil
}

func (p *Player) TakeTurn(ctx context.Context, state *State) error {
	rolls, err := p.roller.RollSync(ctx, 2)
	if err != nil {
		return fmt.Errorf("error rolling: %w", err)
	}

	d1, d2 := int(rolls[0]), int(rolls[1])

	// fmt.Printf("%s: Roll [%d, %d]\n", p.name, d1, d2)

	moves := generateMoves(state.Pawn1Pos, state.Pawn2Pos, d1, d2)
	if len(moves) == 0 {
		return errors.New("no valid moves")
	}

	move := p.strategy.ChooseMove(ctx, *state, moves)
	state.Move(move)

	err = p.peer.SendMove(ctx, move)
	if err != nil {
		return fmt.Errorf("error sending turn: %w", err)
	}

	return nil
}

func validateMove(rolls []uint16, state State, move Move) error {
	d1, d2 := int(rolls[0]), int(rolls[1])

	moves := generateMoves(state.PeerPawn1Pos, state.PeerPawn1Pos, d1, d2)
	if len(moves) == 0 {
		return errors.New("no valid moves")
	}

	if slices.ContainsFunc(moves, func(m Move) bool {
		return m.Pawn1.Result == move.Pawn1.Result
		// return m.Pawn1.Result == move.Pawn1.Result && m.Pawn2.Result == move.Pawn2.Result
	}) {
		return nil
	}
	return fmt.Errorf("other player completed invalid move: %s = %d", move.Pawn1.Expr.String(), move.Pawn1.Result)
}
