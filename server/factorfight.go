package server

import (
	"context"
	"fmt"

	"ztg/dice"
	"ztg/factorfight"
	"ztg/identity"

	factorfightpb "ztg/gen/go/factorfight/v1"
	gamepb "ztg/gen/go/game/v1"
	identitypb "ztg/gen/go/identity/v1"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type FactorFightConfig struct {
	// Strategy controls how your implementation will choose which move to make after rolling
	Strategy factorfight.Strategy
	// OnGameComplete runs after your server receives a challenge from another player. The most basic/common
	// use for it would be notifying yourself of win/lose
	OnGameComplete func(win bool, log factorfight.GameLog)
}

// convertInternalFactorfightMoveToProto converts factorfight.Move to factorfightpb.Move.
func convertInternalFactorfightMoveToProto(move factorfight.Move) *factorfightpb.Move {
	toProtoPawn := func(p factorfight.PawnMove) *factorfightpb.PawnMove {
		return &factorfightpb.PawnMove{
			Result: int32(p.Result),
			Bump:   int32(p.Bump),
			Expr:   p.Expr.String(),
		}
	}
	return &factorfightpb.Move{
		Pawn1: toProtoPawn(move.Pawn1),
		Pawn2: toProtoPawn(move.Pawn2),
	}
}

func convertfactorfightpbMoveToInternal(move *factorfightpb.Move) factorfight.Move {
	// Conversion with safer embedding of Expr.
	return factorfight.Move{
		Pawn1: factorfight.PawnMove{
			Result: int(move.GetPawn1().GetResult()),
			Bump:   factorfight.BumpStatus(move.GetPawn1().GetBump()),
			Expr:   &factorfight.Expr{},
		},
		Pawn2: factorfight.PawnMove{
			Result: int(move.GetPawn2().GetResult()),
			Bump:   factorfight.BumpStatus(move.GetPawn2().GetBump()),
			// TODO: proto encode and decode Expr
			Expr: &factorfight.Expr{},
		},
	}
}

// factorfightPeer implements the Peer interface for use with streaming.
type factorfightPeer struct {
	stream       factorfightStream
	dicePeer     dicePeer
	signedServer *SignedServer
}

var _ factorfight.Peer = factorfightPeer{}

// Dice returns nil as this peer does not use dice.Peer for game communication.
func (p factorfightPeer) Dice() dice.Peer {
	// Note: Update if dice.Peer is required for streaming.
	return p.dicePeer
}

// SendMove sends a move to the stream.
func (p factorfightPeer) SendMove(ctx context.Context, move factorfight.Move) error {
	protoMove := convertInternalFactorfightMoveToProto(move)
	ffMsg := &factorfightpb.FactorFightMessage{
		Message: &factorfightpb.FactorFightMessage_Move{Move: protoMove},
	}
	signedMsg := &factorfightpb.SignedFactorFightMessage{
		Message: ffMsg,
	}

	if p.signedServer != nil {
		var err error
		signedMsg, err = p.signedServer.createSignedFactorFightMessage(ffMsg, nil, 1)
		if err != nil {
			return err
		}
	}

	return p.stream.Send(signedMsg)
}

type factorfightStream interface {
	Recv() (*factorfightpb.SignedFactorFightMessage, error)
	Send(*factorfightpb.SignedFactorFightMessage) error
}

// RecvMove receives a move from the stream.
func (p factorfightPeer) RecvMove(ctx context.Context) (factorfight.Move, error) {
	msg, err := p.stream.Recv()
	if err != nil {
		return factorfight.Move{}, err
	}

	// Verify signature if signedServer exists
	if p.signedServer != nil {
		if msg.Signature == nil {
			return factorfight.Move{}, fmt.Errorf("message is not signed but signedServer is configured")
		}

		msgBytes, err := p.signedServer.serializeMessage(msg.Message)
		if err != nil {
			return factorfight.Move{}, fmt.Errorf("failed to serialize message: %w", err)
		}

		if err := p.signedServer.verifyOrderedSignature(msgBytes, msg.Signature); err != nil {
			return factorfight.Move{}, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	switch m := msg.Message.Message.(type) {
	case *factorfightpb.FactorFightMessage_Move:
		return convertfactorfightpbMoveToInternal(m.Move), nil
	default:
		return factorfight.Move{}, fmt.Errorf("expected move, got different message type")
	}
}

// factorfightService implements the gRPC server for FactorFight.
type factorfightService struct {
	factorfightpb.UnimplementedFactorFightServiceServer

	cfg        FactorFightConfig
	keyManager *identity.KeyManager
	serverAddr string
	signedMode bool
}

// StreamGame handles the gRPC streaming communication.
func (s *factorfightService) Play(stream factorfightpb.FactorFightService_PlayServer) error {
	signedServer := NewSignedServer(s.keyManager, s.serverAddr)
	dicePeer := dicePeer{
		ffStream:     stream,
		signedServer: signedServer,
	}

	factorfightPeer := factorfightPeer{
		stream:       stream,
		dicePeer:     dicePeer,
		signedServer: signedServer,
	}

	strategy := s.cfg.Strategy
	if s.cfg.Strategy != nil {
		strategy = factorfight.DefaultStrategy
	}

	session, err := factorfight.NewSession(factorfightPeer, strategy)
	if err != nil {
		return err
	}

	win, log, err := session.PlayWithInitiative(stream.Context(), true)
	if err != nil {
		return err
	}

	if s.cfg.OnGameComplete != nil {
		s.cfg.OnGameComplete(win, log)
	}

	return nil
}

func playFactorFight(ctx context.Context, conn *grpc.ClientConn, cfg FactorFightConfig, keyManager *identity.KeyManager, serverAddr string) (*gamepb.ChallengeResponse, error) {
	ffClient := factorfightpb.NewFactorFightServiceClient(conn)
	stream, err := ffClient.Play(ctx)
	if err != nil {
		return nil, err
	}

	// Get peer identity for signature verification
	identityClient := identitypb.NewIdentityServiceClient(conn)
	peerIdentity, err := identityClient.GetIdentity(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("failed to get peer identity: %w", err)
	}

	signedServer := NewSignedServer(keyManager, serverAddr)
	// Cache the peer's public key for signature verification
	signedServer.AddPeerIdentity(conn.Target(), peerIdentity.PublicKey)
	dicePeer := dicePeer{
		ffStream:     stream,
		signedServer: signedServer,
	}

	factorfightPeer := factorfightPeer{
		stream:       stream,
		dicePeer:     dicePeer,
		signedServer: signedServer,
	}

	strategy := cfg.Strategy
	if cfg.Strategy != nil {
		strategy = factorfight.DefaultStrategy
	}

	session, err := factorfight.NewSession(factorfightPeer, strategy)
	if err != nil {
		return nil, err
	}

	win, log, err := session.PlayWithInitiative(stream.Context(), false)
	if err != nil {
		return nil, err
	}

	if cfg.OnGameComplete != nil {
		cfg.OnGameComplete(win, log)
	}

	err = stream.CloseSend()
	return &gamepb.ChallengeResponse{
		Win: &win,
	}, err
}
