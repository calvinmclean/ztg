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
	pos, peerPos := 0, 0

	if !goFirst {
		var err error
		peerPos, err = p.OtherTurn(ctx, peerPos)
		if err != nil {
			return false, fmt.Errorf("error doing other player's first turn: %w", err)
		}
	}

	turnNum := 0
	for {
		// fmt.Printf("%s (%d): Taking turn\n", p.name, turnNum)
		newPos, err := p.TakeTurn(ctx, pos, turnNum)
		if err != nil {
			return false, fmt.Errorf("error taking turn: %w", err)
		}

		// fmt.Printf("%s (%d): Moving %d -> %d\n", p.name, turnNum, pos, newPos)
		pos = newPos
		if pos == goal {
			return true, nil
		}

		peerPos, err = p.OtherTurn(ctx, peerPos)
		if err != nil {
			return false, fmt.Errorf("error doing other player's first turn: %w", err)
		}

		if peerPos == goal {
			return false, nil
		}

		turnNum++
	}
}

func validateTurn(rolls []uint16, start, end int) error {
	d1, d2 := int(rolls[0]), int(rolls[1])

	moves := validMoves(start, d1, d2)
	if len(moves) == 0 {
		return errors.New("no valid moves")
	}

	if slices.Contains(moves, end) {
		return nil
	}
	return fmt.Errorf("other player completed invalid move: [%d, %d] %d -> %d", d1, d2, start, end)
}

func (p *Player) OtherTurn(ctx context.Context, pos int) (int, error) {
	rolls, err := p.roller.RollSync(ctx, 2)
	if err != nil {
		return 0, fmt.Errorf("error rolling: %w", err)
	}

	peerTurn, err := p.peer.RecvTurn(ctx)
	if err != nil {
		return 0, fmt.Errorf("error receiving turn: %w", err)
	}

	err = validateTurn(rolls, pos, peerTurn.Pawn1Pos)
	if err != nil {
		return 0, err
	}

	return peerTurn.Pawn1Pos, nil
}

func (p *Player) TakeTurn(ctx context.Context, pos, i int) (int, error) {
	rolls, err := p.roller.RollSync(ctx, 2)
	if err != nil {
		return 0, fmt.Errorf("error rolling: %w", err)
	}

	d1, d2 := int(rolls[0]), int(rolls[1])

	// fmt.Printf("%s (%d): Roll [%d, %d]\n", p.name, i, d1, d2)

	moves := validMoves(pos, d1, d2)
	if len(moves) == 0 {
		return 0, errors.New("no valid moves")
	}

	newPos := p.strategy.ChooseMove(ctx, moves)

	err = p.peer.SendTurn(ctx, Turn{Pawn1Pos: newPos})
	if err != nil {
		return 0, fmt.Errorf("error sending turn: %w", err)
	}

	return newPos, nil
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
