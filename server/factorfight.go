package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ztg/dice"
	"ztg/factorfight"
	"ztg/identity"
	ffproto "ztg/proto/factorfight"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FactorfightPeer implements a Peer that works over a gRPC connection
type FactorfightPeer struct {
	id       identity.Identity
	dicePeer *DicePeer
	client   ffproto.FactorFightServiceClient

	in        chan factorfight.Move
	sessionID string
}

var _ factorfight.Peer = &FactorfightPeer{}

// NewFactorfightPeer creates a new GRPCPeer with the given gRPC client.
func NewFactorfightPeer(id identity.Identity, dicePeer *DicePeer, client ffproto.FactorFightServiceClient, in chan factorfight.Move, sessionID string) (*FactorfightPeer, error) {
	return &FactorfightPeer{
		dicePeer:  dicePeer,
		client:    client,
		in:        in,
		id:        id,
		sessionID: sessionID,
	}, nil
}

func (p *FactorfightPeer) SetClient(client ffproto.FactorFightServiceClient) {
	p.client = client
}

// Dice returns the underlying Peer implementation.
func (p *FactorfightPeer) Dice() dice.Peer {
	return p.dicePeer
}

func (p FactorfightPeer) Identity() identity.Identity {
	return p.id
}

// SendMove sends a Move as part of the Peer interface using the client.
func (p *FactorfightPeer) SendMove(ctx context.Context, move factorfight.Move) error {
	for range retries {
		_, err := p.client.ReceiveMove(ctx, &ffproto.Move{
			Pawn1: &ffproto.PawnMove{
				Result: int32(move.Pawn1.Result),
				Bump:   int32(move.Pawn1.Bump),
				Expr:   move.Pawn1.Expr.String(),
			},
			Pawn2: &ffproto.PawnMove{
				Result: int32(move.Pawn2.Result),
				Bump:   int32(move.Pawn2.Bump),
				Expr:   move.Pawn2.Expr.String(),
			},
			SessionId: p.sessionID,
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
func (p *FactorfightPeer) RecvMove(ctx context.Context) (factorfight.Move, error) {
	select {
	case move := <-p.in:
		return move, nil
	case <-ctx.Done():
		return factorfight.Move{}, ctx.Err()
	}
}

// Utility function to parse an expression string into an Expr object.
func parseExpr(exprStr string) *factorfight.Expr {
	// NOTE: Implement expression parsing logic here.
	// For now, return a placeholder.
	return &factorfight.Expr{}
}

type FactorfightServer struct {
	ffproto.UnimplementedFactorFightServiceServer
	sessionMap *sync.Map
}

func NewFactorfightServer(sessionMap *sync.Map) *FactorfightServer {
	return &FactorfightServer{sessionMap: sessionMap}
}

func (p *FactorfightServer) ReceiveMove(ctx context.Context, incoming *ffproto.Move) (*ffproto.Empty, error) {
	move := factorfight.Move{
		Pawn1: factorfight.PawnMove{
			Result: int(incoming.Pawn1.Result),
			Bump:   factorfight.BumpStatus(incoming.Pawn1.Bump),
			Expr:   parseExpr(incoming.Pawn1.Expr),
		},
		Pawn2: factorfight.PawnMove{
			Result: int(incoming.Pawn2.Result),
			Bump:   factorfight.BumpStatus(incoming.Pawn2.Bump),
			Expr:   parseExpr(incoming.Pawn2.Expr),
		},
	}

	session, ok := p.sessionMap.Load(incoming.SessionId)
	if !ok {
		return nil, fmt.Errorf("session %s not found", incoming.SessionId)
	}
	sessionChan, ok := session.(chan factorfight.Move)
	if !ok {
		return nil, fmt.Errorf("incorrect type for session %s: %T", incoming.SessionId, session)
	}

	sessionChan <- move
	return &ffproto.Empty{}, nil
}
