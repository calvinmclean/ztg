package dice

import (
	"context"
	"time"

	"ztg/proto/dice"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCPeer implements a Peer that works over a gRPC connection
type GRPCPeer struct {
	dice.UnimplementedRollerServiceServer

	client dice.RollerServiceClient // gRPC client for communicating with remote peers
	in     chan Message             // Channel for receiving messages from gRPC server
}

var _ Peer = &GRPCPeer{}

// NewGRPCPeer creates a new GRPCPeer with the given gRPC client.
func NewGRPCPeer(client dice.RollerServiceClient) (*GRPCPeer, error) {
	return &GRPCPeer{
		client: client,
		in:     make(chan Message, 1),
	}, nil
}

func (p *GRPCPeer) SetClient(client dice.RollerServiceClient) {
	p.client = client
}

// Send sends a message to the remote gRPC server using the gRPC client.
func (p *GRPCPeer) Send(ctx context.Context, msg Message) error {
	for range retries {
		_, err := p.client.ReceiveMessage(ctx, &dice.Message{
			MsgType: dice.MessageType(msg.Type),
			Data:    msg.Data[:],
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
func (p *GRPCPeer) Recv(ctx context.Context) (Message, error) {
	select {
	case msg := <-p.in:
		return msg, nil
	case <-ctx.Done():
		return Message{}, ctx.Err()
	}
}

// ReceiveMessage handles incoming messages from gRPC clients.
func (p *GRPCPeer) ReceiveMessage(ctx context.Context, incoming *dice.Message) (*dice.Empty, error) {
	msg := Message{
		Type: MsgType(incoming.MsgType),
		Data: toFixed32(incoming.Data),
	}
	p.in <- msg
	return &dice.Empty{}, nil
}

// Helper function to copy data to [32]byte for Message.
func toFixed32(data []byte) [32]byte {
	var fixed [32]byte
	copy(fixed[:], data)
	return fixed
}
