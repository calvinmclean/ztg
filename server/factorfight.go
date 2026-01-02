package server

import (
	"context"
	"fmt"

	"ztg/dice"
	"ztg/factorfight"

	factorfightpb "ztg/gen/go/factorfight/v1"
	gamepb "ztg/gen/go/game/v1"

	"google.golang.org/grpc"
)

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
	stream   factorfightStream
	dicePeer dicePeer
}

var _ factorfight.Peer = factorfightPeer{}

// Dice returns nil as this peer does not use dice.Peer for game communication.
func (p factorfightPeer) Dice() dice.Peer {
	// Note: Update if dice.Peer is required for streaming.
	return p.dicePeer
}

// SendMove sends a move to the stream.
func (p factorfightPeer) SendMove(ctx context.Context, move factorfight.Move) error {
	return p.stream.Send(&factorfightpb.FactorFightMessage{
		Message: &factorfightpb.FactorFightMessage_Move{Move: convertInternalFactorfightMoveToProto(move)},
	})
}

type factorfightStream interface {
	Recv() (*factorfightpb.FactorFightMessage, error)
	Send(*factorfightpb.FactorFightMessage) error
}

// RecvMove receives a move from the stream.
func (p factorfightPeer) RecvMove(ctx context.Context) (factorfight.Move, error) {
	msg, err := p.stream.Recv()
	if err != nil {
		return factorfight.Move{}, err
	}

	switch m := msg.Message.(type) {
	case *factorfightpb.FactorFightMessage_Move:
		return convertfactorfightpbMoveToInternal(m.Move), nil
	default:
		return factorfight.Move{}, fmt.Errorf("expected move, got different message type")
	}
}

// factorfightService implements the gRPC server for FactorFight.
type factorfightService struct {
	factorfightpb.UnimplementedFactorFightServiceServer
}

// StreamGame handles the gRPC streaming communication.
func (s *factorfightService) Play(stream factorfightpb.FactorFightService_PlayServer) error {
	dicePeer := dicePeer{
		ffStream: stream,
	}

	factorfightPeer := factorfightPeer{
		stream:   stream,
		dicePeer: dicePeer,
	}

	session, err := factorfight.NewSession(factorfightPeer, factorfight.DefaultStrategy)
	if err != nil {
		return err
	}

	_, log, err := session.PlayWithInitiative(stream.Context(), true)
	if err != nil {
		return err
	}

	fmt.Println(log)

	return nil
}

func playFactorFight(ctx context.Context, conn *grpc.ClientConn) (*gamepb.ChallengeResponse, error) {
	ffClient := factorfightpb.NewFactorFightServiceClient(conn)
	stream, err := ffClient.Play(ctx)
	if err != nil {
		return nil, err
	}

	dicePeer := dicePeer{
		ffStream: stream,
	}

	factorfightPeer := factorfightPeer{
		stream:   stream,
		dicePeer: dicePeer,
	}

	session, err := factorfight.NewSession(factorfightPeer, factorfight.DefaultStrategy)
	if err != nil {
		return nil, err
	}

	win, log, err := session.PlayWithInitiative(stream.Context(), false)
	if err != nil {
		return nil, err
	}

	fmt.Println(log)

	err = stream.CloseSend()
	return &gamepb.ChallengeResponse{
		Win: &win,
	}, err
}
