package factorfight

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"ztg/dice"
)

// TODO: Handle "bump".
// I need to consider the order of moves because if p2 is on 3 and p1 moves 1->3, p2 is bumped. If I move p2 first, then it's fine.
// I also want to indicate on the Move struct that bumps are happening

// TODO: initialize game between players (who goes first?)

const goal = 101

// GameError wraps an error with the game log
type GameError struct {
	GameLog GameLog
	Err     error
}

func (ge *GameError) Error() string {
	return ge.Err.Error()
}

func (ge *GameError) Unwrap() error {
	return ge.Err
}

type Peer interface {
	SendMove(context.Context, Move) error
	RecvMove(context.Context) (Move, error)
	Dice() dice.Peer
}

// GameLogEntry represents a single move in the game log
type GameLogEntry struct {
	Player    string
	Move      Move
	TurnNum   int
	IsPeer    bool
	FromState State
	Rolls     [2]uint16
}

// GameLog is a type alias for a slice of GameLogEntry
type GameLog []GameLogEntry

// State represents the current state of the game
type State struct {
	Pawn1Pos     int
	Pawn2Pos     int
	PeerPawn1Pos int
	PeerPawn2Pos int
	GameLog      GameLog // Log of all moves in the game
}

// AddGameLogEntry creates and adds a game log entry
func (s *State) AddGameLogEntry(playerName string, move Move, turnNum int, isPeer bool, rolls []uint16) {
	var rollsArray [2]uint16
	if len(rolls) >= 2 {
		rollsArray = [2]uint16{rolls[0], rolls[1]}
	}

	logEntry := GameLogEntry{
		Player:  playerName,
		Move:    move,
		TurnNum: turnNum,
		IsPeer:  isPeer,
		Rolls:   rollsArray,
		FromState: State{
			Pawn1Pos:     s.Pawn1Pos,
			Pawn2Pos:     s.Pawn2Pos,
			PeerPawn1Pos: s.PeerPawn1Pos,
			PeerPawn2Pos: s.PeerPawn2Pos,
			GameLog:      nil,
		},
	}

	s.GameLog = append(s.GameLog, logEntry)
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

func (s State) Win() bool {
	return s.Pawn1Pos == goal && s.Pawn2Pos == goal
}

func (s State) Lose() bool {
	return s.PeerPawn1Pos == goal && s.PeerPawn2Pos == goal
}

// String returns a formatted string representation of the game log
func (gl GameLog) String() string {
	if len(gl) == 0 {
		return "No moves logged yet."
	}

	var result string
	result += fmt.Sprintf("Game Log (%d moves):\n", len(gl))
	result += "================\n"

	for _, entry := range gl {
		player := "Peer"
		if !entry.IsPeer {
			player = entry.Player
		}

		result += fmt.Sprintf("Turn %d - %s:\n", entry.TurnNum, player)
		result += fmt.Sprintf("  Rolls: [%d, %d]\n", entry.Rolls[0], entry.Rolls[1])
		result += fmt.Sprintf("  Pawn1: %s → %d\n", entry.Move.Pawn1.Expr.String(), entry.Move.Pawn1.Result)
		result += fmt.Sprintf("  Pawn2: %s → %d\n", entry.Move.Pawn2.Expr.String(), entry.Move.Pawn2.Result)
		result += fmt.Sprintf("  From: P1=%d, P2=%d | PeerP1=%d, PeerP2=%d\n",
			entry.FromState.Pawn1Pos, entry.FromState.Pawn2Pos,
			entry.FromState.PeerPawn1Pos, entry.FromState.PeerPawn2Pos)
		result += "\n"
	}

	return result
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

func (p *Player) Play(ctx context.Context, goFirst bool) (bool, GameLog, error) {
	state := &State{}

	if !goFirst {
		err := p.OtherTurn(ctx, state, 0)
		if err != nil {
			return false, state.GameLog, &GameError{
				GameLog: state.GameLog,
				Err:     fmt.Errorf("error doing other player's first turn: %w", err),
			}
		}
	}

	turnNum := 0
	for {
		// fmt.Printf("%s (%d): Taking turn\n", p.name, turnNum)
		err := p.TakeTurn(ctx, state, turnNum)
		if err != nil {
			return false, state.GameLog, &GameError{
				GameLog: state.GameLog,
				Err:     fmt.Errorf("error taking turn: %w", err),
			}
		}

		// fmt.Printf("%s (%d): Moving %d -> %d\n", p.name, turnNum, pos, newPos)
		if state.Win() {
			return true, state.GameLog, nil
		}

		err = p.OtherTurn(ctx, state, turnNum)
		if err != nil {
			return false, state.GameLog, &GameError{
				GameLog: state.GameLog,
				Err:     fmt.Errorf("error doing other player's first turn: %w", err),
			}
		}

		if state.Lose() {
			return false, state.GameLog, nil
		}

		turnNum++
	}
}

func (p *Player) OtherTurn(ctx context.Context, state *State, turnNum int) error {
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

	state.AddGameLogEntry("Peer", peerMove, turnNum, true, rolls)
	state.PeerMove(peerMove)

	return nil
}

func (p *Player) TakeTurn(ctx context.Context, state *State, turnNum int) error {
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
	rollsSlice := []uint16{uint16(d1), uint16(d2)}
	state.AddGameLogEntry(p.name, move, turnNum, false, rollsSlice)
	state.Move(move)
	// fmt.Printf("%s: %s = %d | %s = %d\n", p.name, move.Pawn1.Expr.String(), move.Pawn1.Result, move.Pawn2.Expr.String(), move.Pawn2.Result)

	err = p.peer.SendMove(ctx, move)
	if err != nil {
		return fmt.Errorf("error sending turn: %w", err)
	}

	return nil
}

func validateMove(rolls []uint16, state State, move Move) error {
	d1, d2 := int(rolls[0]), int(rolls[1])

	moves := generateMoves(state.PeerPawn1Pos, state.PeerPawn2Pos, d1, d2)
	if len(moves) == 0 {
		return errors.New("no valid moves")
	}

	if slices.ContainsFunc(moves, func(m Move) bool {
		return m.Pawn1.Result == move.Pawn1.Result && m.Pawn2.Result == move.Pawn2.Result
	}) {
		return nil
	}
	return fmt.Errorf("other player completed invalid move: %s = %d", move.Pawn1.Expr.String(), move.Pawn1.Result)
}
