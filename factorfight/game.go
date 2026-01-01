package factorfight

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"ztg/dice"
)

// TODO: implement state validation step? This could be useful for detecting bump errors if one player doesn't move a bump correctly
// Or maybe this type of thing will always be caught by invalid moves?

const goal = 101

// Session holds the Peer and Roller instances for a game.
type Session struct {
	Peer     Peer
	Roller   dice.Roller
	Strategy Strategy
}

func NewSession(peer Peer, strategy Strategy) (Session, error) {
	roller, err := dice.NewRoller(10, peer.Dice())
	if err != nil {
		return Session{}, fmt.Errorf("error creating roller: %w", err)
	}

	return Session{
		Roller:   roller,
		Peer:     peer,
		Strategy: strategy,
	}, nil
}

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
	State     State
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
func (s *State) AddGameLogEntry(move Move, turnNum int, isPeer bool, rolls []uint16) {
	var rollsArray [2]uint16
	if len(rolls) >= 2 {
		rollsArray = [2]uint16{rolls[0], rolls[1]}
	}

	newState := *s
	if isPeer {
		newState.PeerMove(move)
	} else {
		newState.Move(move)
	}

	name := "Self"
	if isPeer {
		name = "Peer"
	}

	logEntry := GameLogEntry{
		Player:  name,
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
		State: State{
			Pawn1Pos:     newState.Pawn1Pos,
			Pawn2Pos:     newState.Pawn2Pos,
			PeerPawn1Pos: newState.PeerPawn1Pos,
			PeerPawn2Pos: newState.PeerPawn2Pos,
			GameLog:      nil,
		},
	}

	s.GameLog = append(s.GameLog, logEntry)
}

// Move updates the pawn positions according to the Move
func (s *State) Move(m Move) {
	s.Pawn1Pos = m.Pawn1.Result
	s.Pawn2Pos = m.Pawn2.Result

	// Handle possible Bumps
	switch m.Pawn1.Bump {
	case BumpedPeer1:
		s.PeerPawn1Pos = 0
	case BumpedPeer2:
		s.PeerPawn2Pos = 0
	case BumpedSelf:
		s.Pawn1Pos = 0
	}

	switch m.Pawn2.Bump {
	case BumpedPeer1:
		s.PeerPawn1Pos = 0
	case BumpedPeer2:
		s.PeerPawn2Pos = 0
	case BumpedSelf:
		s.Pawn1Pos = 0
	}
}

// PeerMove updates the peer's pawn positions according to the Move
func (s *State) PeerMove(m Move) {
	s.PeerPawn1Pos = m.Pawn1.Result
	s.PeerPawn2Pos = m.Pawn2.Result

	// Handle possible Bumps
	switch m.Pawn1.Bump {
	case BumpedPeer1:
		s.Pawn1Pos = 0
	case BumpedPeer2:
		s.Pawn2Pos = 0
	case BumpedSelf:
		s.PeerPawn1Pos = 0
	}

	switch m.Pawn2.Bump {
	case BumpedPeer1:
		s.Pawn1Pos = 0
	case BumpedPeer2:
		s.Pawn2Pos = 0
	case BumpedSelf:
		s.PeerPawn1Pos = 0
	}
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
		player := entry.Player
		if entry.IsPeer {
			player += " (peer)"
		}

		result += fmt.Sprintf("Turn %d - %s:\n", entry.TurnNum, player)
		result += fmt.Sprintf("  Rolls: [%d, %d]\n", entry.Rolls[0], entry.Rolls[1])
		result += fmt.Sprintf("  Pawn1: %s → %d (%s)\n", entry.Move.Pawn1.Expr.String(), entry.Move.Pawn1.Result, entry.Move.Pawn1.Bump.String())
		result += fmt.Sprintf("  Pawn2: %s → %d (%s)\n", entry.Move.Pawn2.Expr.String(), entry.Move.Pawn2.Result, entry.Move.Pawn2.Bump.String())
		result += fmt.Sprintf("   From: P1=%d, P2=%d | PeerP1=%d, PeerP2=%d\n",
			entry.FromState.Pawn1Pos, entry.FromState.Pawn2Pos,
			entry.FromState.PeerPawn1Pos, entry.FromState.PeerPawn2Pos)
		result += fmt.Sprintf("  State: P1=%d, P2=%d | PeerP1=%d, PeerP2=%d\n",
			entry.State.Pawn1Pos, entry.State.Pawn2Pos,
			entry.State.PeerPawn1Pos, entry.State.PeerPawn2Pos)
		result += "\n"
	}

	if gl[len(gl)-1].State.Win() {
		result += "Win!\n"
	} else {
		result += "Lose!\n"
	}

	return result
}

// PlayWithInitiative will first roll initiative to decide who goes first
func (s *Session) PlayWithInitiative(ctx context.Context, high bool) (bool, GameLog, error) {
	first, err := s.RollInitiative(ctx, high)
	if err != nil {
		return false, GameLog{}, fmt.Errorf("error rolling initiative: %w", err)
	}

	return s.Play(ctx, first)
}

func (s *Session) Play(ctx context.Context, goFirst bool) (bool, GameLog, error) {
	state := &State{}

	if !goFirst {
		err := s.OtherTurn(ctx, state, 0)
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
		err := s.TakeTurn(ctx, state, turnNum)
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

		err = s.OtherTurn(ctx, state, turnNum)
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

func (s *Session) OtherTurn(ctx context.Context, state *State, turnNum int) error {
	rolls, err := s.Roller.RollSync(ctx, 2)
	if err != nil {
		return fmt.Errorf("error rolling: %w", err)
	}

	peerMove, err := s.Peer.RecvMove(ctx)
	if err != nil {
		return fmt.Errorf("error receiving turn: %w", err)
	}

	err = validateMove(rolls, *state, peerMove)
	if err != nil {
		return err
	}

	state.AddGameLogEntry(peerMove, turnNum, true, rolls)
	state.PeerMove(peerMove)

	return nil
}

func (s *Session) TakeTurn(ctx context.Context, state *State, turnNum int) error {
	rolls, err := s.Roller.RollSync(ctx, 2)
	if err != nil {
		return fmt.Errorf("error rolling: %w", err)
	}

	d1, d2 := int(rolls[0]), int(rolls[1])

	// fmt.Printf("%s: Roll [%d, %d]\n", p.name, d1, d2)

	moves := generateMoves(state.Pawn1Pos, state.Pawn2Pos, d1, d2, state.PeerPawn1Pos, state.PeerPawn2Pos)
	if len(moves) == 0 {
		return errors.New("no valid moves")
	}

	move := s.Strategy.ChooseMove(ctx, *state, moves)
	rollsSlice := []uint16{uint16(d1), uint16(d2)}
	state.AddGameLogEntry(move, turnNum, false, rollsSlice)
	state.Move(move)
	// fmt.Printf("%s: %s = %d | %s = %d\n", p.name, move.Pawn1.Expr.String(), move.Pawn1.Result, move.Pawn2.Expr.String(), move.Pawn2.Result)

	err = s.Peer.SendMove(ctx, move)
	if err != nil {
		return fmt.Errorf("error sending turn: %w", err)
	}

	return nil
}

func validateMove(rolls []uint16, state State, move Move) error {
	d1, d2 := int(rolls[0]), int(rolls[1])

	moves := generateMoves(state.PeerPawn1Pos, state.PeerPawn2Pos, d1, d2, state.Pawn1Pos, state.Pawn2Pos)
	if len(moves) == 0 {
		return errors.New("no valid moves")
	}

	if !slices.ContainsFunc(moves, func(m Move) bool {
		return m.Pawn1.Result == move.Pawn1.Result && m.Pawn2.Result == move.Pawn2.Result &&
			m.Pawn1.Bump == move.Pawn1.Bump && m.Pawn2.Bump == move.Pawn2.Bump
	}) {
		return fmt.Errorf("other player completed invalid move: %s = %d", move.Pawn1.Expr.String(), move.Pawn1.Result)
	}
	return nil
}

// RollInitiative rolls a 10-sided die to determine who goes first. If high is true, this player wants 6-10, otherwise 1-5.
// Returns true if going first,
func (s Session) RollInitiative(ctx context.Context, high bool) (bool, error) {
	rolls, err := s.Roller.RollSync(ctx, 1)
	if err != nil {
		return false, fmt.Errorf("error rolling: %w", err)
	}
	if len(rolls) != 1 {
		return false, fmt.Errorf("unexpected number of rolls: %d", len(rolls))
	}

	if high {
		return rolls[0] > 5, nil
	}

	return rolls[0] <= 5, nil
}
