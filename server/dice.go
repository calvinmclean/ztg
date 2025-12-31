package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ztg/dice"
	protodice "ztg/proto/dice"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	retries    = 5
	retryDelay = 500 * time.Millisecond
)

// DicePeer implements a Peer that works over a gRPC connection
type DicePeer struct {
	protodice.UnimplementedRollerServiceServer

	client protodice.RollerServiceClient // gRPC client for communicating with remote peers
	in     chan dice.Message             // Channel for receiving messages from gRPC server

	sessionID string
}

var _ dice.Peer = &DicePeer{}

// NewDicePeer creates a new GRPCPeer with the given gRPC client.
func NewDicePeer(client protodice.RollerServiceClient, in chan dice.Message, sessionID string) (*DicePeer, error) {
	return &DicePeer{
		client:    client,
		in:        in,
		sessionID: sessionID,
	}, nil
}

func (p *DicePeer) SetClient(client protodice.RollerServiceClient) {
	p.client = client
}

// Send sends a message to the remote gRPC server using the gRPC client.
func (p *DicePeer) Send(ctx context.Context, msg dice.Message) error {
	for range retries {
		_, err := p.client.ReceiveMessage(ctx, &protodice.Message{
			SessionId: p.sessionID,
			MsgType:   protodice.MessageType(msg.Type),
			Data:      msg.Data[:],
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

// Recv receives a message from the GRPC server by reading from the channel.
func (p *DicePeer) Recv(ctx context.Context) (dice.Message, error) {
	select {
	case msg := <-p.in:
		return msg, nil
	case <-ctx.Done():
		return dice.Message{}, ctx.Err()
	}
}

// Helper function to copy data to [32]byte for Message.
func toFixed32(data []byte) [32]byte {
	var fixed [32]byte
	copy(fixed[:], data)
	return fixed
}

type DiceServer struct {
	protodice.UnimplementedRollerServiceServer
	sessionMap *sync.Map
}

func NewDiceServer(sessionMap *sync.Map) *DiceServer {
	return &DiceServer{sessionMap: sessionMap}
}

func (p *DiceServer) ReceiveMessage(ctx context.Context, incoming *protodice.Message) (*protodice.Empty, error) {
	msg := dice.Message{
		Type: dice.MsgType(incoming.MsgType),
		Data: toFixed32(incoming.Data),
	}

	session, ok := p.sessionMap.Load(incoming.SessionId)
	if !ok {
		return nil, fmt.Errorf("session %s not found", incoming.SessionId)
	}
	sessionChan, ok := session.(chan dice.Message)
	if !ok {
		return nil, fmt.Errorf("incorrect type for session %s: %T", incoming.SessionId, session)
	}

	sessionChan <- msg
	return &protodice.Empty{}, nil
}
