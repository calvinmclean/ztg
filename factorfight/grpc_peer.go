package factorfight

import (
	"context"
	"time"

	"ztg/dice"
	"ztg/proto/factorfight"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	retries    = 5
	retryDelay = 500 * time.Millisecond
)

// GRPCPeer implements a Peer that works over a gRPC connection
type GRPCPeer struct {
	factorfight.UnimplementedFactorFightServiceServer

	dicePeer *dice.GRPCPeer
	client   factorfight.FactorFightServiceClient

	in chan Move
}

var _ Peer = &GRPCPeer{}

// NewGRPCPeer creates a new GRPCPeer with the given gRPC client.
func NewGRPCPeer(dicePeer *dice.GRPCPeer, client factorfight.FactorFightServiceClient) (*GRPCPeer, error) {
	return &GRPCPeer{
		dicePeer: dicePeer,
		client:   client,
		in:       make(chan Move, 1),
	}, nil
}

func (p *GRPCPeer) SetClient(client factorfight.FactorFightServiceClient) {
	p.client = client
}

// Dice returns the underlying Peer implementation.
func (p *GRPCPeer) Dice() dice.Peer {
	return p.dicePeer
}

// SendMove sends a Move as part of the Peer interface using the client.
func (p *GRPCPeer) SendMove(ctx context.Context, move Move) error {
	for range retries {
		_, err := p.client.ReceiveMove(ctx, &factorfight.Move{
			Pawn1: &factorfight.PawnMove{
				Result: int32(move.Pawn1.Result),
				Bump:   int32(move.Pawn1.Bump),
				Expr:   move.Pawn1.Expr.String(),
			},
			Pawn2: &factorfight.PawnMove{
				Result: int32(move.Pawn2.Result),
				Bump:   int32(move.Pawn2.Bump),
				Expr:   move.Pawn2.Expr.String(),
			},
		})

		switch status.Code(err) {
		case codes.OK:
			return nil
		case codes.Unavailable:
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		if err != nil {
			return err
		}
	}
	return nil
}

// RecvMove retrieves incoming moves as part of the Peer interface.
func (p *GRPCPeer) RecvMove(ctx context.Context) (Move, error) {
	select {
	case move := <-p.in:
		return move, nil
	case <-ctx.Done():
		return Move{}, ctx.Err()
	}
}

// ReceiveMove handles incoming moves from gRPC clients.
func (p *GRPCPeer) ReceiveMove(ctx context.Context, incoming *factorfight.Move) (*factorfight.Empty, error) {
	move := Move{
		Pawn1: PawnMove{
			Result: int(incoming.Pawn1.Result),
			Bump:   BumpStatus(incoming.Pawn1.Bump),
			Expr:   parseExpr(incoming.Pawn1.Expr),
		},
		Pawn2: PawnMove{
			Result: int(incoming.Pawn2.Result),
			Bump:   BumpStatus(incoming.Pawn2.Bump),
			Expr:   parseExpr(incoming.Pawn2.Expr),
		},
	}
	p.in <- move
	return &factorfight.Empty{}, nil
}

// Utility function to parse an expression string into an Expr object.
func parseExpr(exprStr string) *Expr {
	// NOTE: Implement expression parsing logic here.
	// For now, return a placeholder.
	return &Expr{}
}
