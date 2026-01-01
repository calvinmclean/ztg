package server

import (
	"context"
	"fmt"

	"ztg/dice"

	dicepb "ztg/gen/proto/dice"
	factorfightpb "ztg/gen/proto/factorfight"
	gamepb "ztg/gen/proto/game"

	"google.golang.org/grpc"
)

// convertdicepbMessageToInternal converts dicepb.Message to dice.Message.
func convertdicepbMessageToInternal(msg *dicepb.Message) dice.Message {
	return dice.Message{
		Type: dice.MsgType(msg.MsgType),
		Data: func(data []byte) [32]byte {
			var fixed [32]byte
			copy(fixed[:], data[:])
			return fixed
		}(msg.Data),
	}
}

// convertInternalDiceMessageToProto converts dice.Message to dicepb.Message.
func convertInternalDiceMessageToProto(msg dice.Message) *dicepb.Message {
	return &dicepb.Message{
		MsgType: dicepb.MessageType(msg.Type),
		Data:    msg.Data[:],
	}
}

type dicePeer struct {
	ffStream factorfightStream
	// TODO: I might be able to make this more generic and combine the two streams since they
	// theoretically both use dice.Message proto
	diceStream interface {
		Recv() (*dicepb.Message, error)
		Send(*dicepb.Message) error
	}
}

var _ dice.Peer = dicePeer{}

// Send sends a message to the stream.
func (p dicePeer) Send(ctx context.Context, msg dice.Message) error {
	if p.diceStream != nil {
		return p.diceStream.Send(convertInternalDiceMessageToProto(msg))
	}
	return p.ffStream.Send(&factorfightpb.FactorFightMessage{
		Message: &factorfightpb.FactorFightMessage_DiceMsg{DiceMsg: convertInternalDiceMessageToProto(msg)},
	})
}

// Recv receives a message from the stream.
func (p dicePeer) Recv(ctx context.Context) (dice.Message, error) {
	if p.diceStream != nil {
		msg, err := p.diceStream.Recv()
		if err != nil {
			return dice.Message{}, err
		}

		return convertdicepbMessageToInternal(msg), nil
	}

	msg, err := p.ffStream.Recv()
	if err != nil {
		return dice.Message{}, err
	}

	switch m := msg.Message.(type) {
	case *factorfightpb.FactorFightMessage_DiceMsg:
		return convertdicepbMessageToInternal(m.DiceMsg), nil
	default:
		return dice.Message{}, fmt.Errorf("expected dice message, got different type")
	}
}

// diceService implements the gRPC server for Dice.
type diceService struct {
	dicepb.UnimplementedDiceServiceServer
}

// StreamGame handles the gRPC streaming communication.
func (s *diceService) Roll(stream dicepb.DiceService_RollServer) error {
	dicePeer := dicePeer{
		diceStream: stream,
	}

	roller, err := dice.NewRoller(10, dicePeer)
	if err != nil {
		return err
	}

	_, err = roller.RollSync(stream.Context(), 1)
	if err != nil {
		return err
	}

	return nil
}

func playHighRoll(ctx context.Context, conn *grpc.ClientConn) (*gamepb.ChallengeResponse, error) {
	client := dicepb.NewDiceServiceClient(conn)

	stream, err := client.Roll(ctx)
	if err != nil {
		return nil, err
	}

	dicePeer := dicePeer{
		diceStream: stream,
	}

	// TODO: add configurable die size
	// TODO: each player rolls a dice and compares to win

	roller, err := dice.NewRoller(10, dicePeer)
	if err != nil {
		return nil, err
	}

	roll, err := roller.RollSync(stream.Context(), 2)
	if err != nil {
		return nil, err
	}

	// 0 is me and 1 is opponent
	win := roll[0] > roll[1]
	var msg string
	if roll[0] == roll[1] {
		msg = "Draw!"
	}

	err = stream.CloseSend()
	return &gamepb.ChallengeResponse{
		Win:     &win,
		Message: msg,
	}, err
}
