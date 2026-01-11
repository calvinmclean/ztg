package server

import (
	"context"
	"fmt"
	"time"

	"ztg/dice"
	"ztg/identity"

	dicepb "ztg/gen/go/dice/v1"
	factorfightpb "ztg/gen/go/factorfight/v1"
	gamepb "ztg/gen/go/game/v1"

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

// dicePeer uses the ffStream OR diceStream to implement a dicePeer. This allows it to be used for FactorFight or just a plain dice game
type dicePeer struct {
	ffStream factorfightStream
	// TODO: I might be able to make this more generic and combine the two streams since they
	// theoretically both use dice.Message proto
	diceStream interface {
		Recv() (*dicepb.SignedMessage, error)
		Send(*dicepb.SignedMessage) error
	}
	signer   *Signer
	verifier *Verifier
}

var _ dice.Peer = dicePeer{}

// Send sends a message to the stream.
func (p dicePeer) Send(ctx context.Context, msg dice.Message) error {
	if p.diceStream != nil {
		protoMsg := convertInternalDiceMessageToProto(msg)
		signedMsg := &dicepb.SignedMessage{
			Message: protoMsg,
		}

		if p.signer != nil {
			var err error
			signedMsg, err = createSignedMessage(&dicepb.SignedMessage{}, p.signer, protoMsg)
			if err != nil {
				return err
			}
		}

		return p.diceStream.Send(signedMsg)
	}

	protoMsg := convertInternalDiceMessageToProto(msg)
	ffMsg := &factorfightpb.FactorFightMessage{
		Message: &factorfightpb.FactorFightMessage_DiceMsg{DiceMsg: protoMsg},
	}
	signedFFMsg := &factorfightpb.SignedFactorFightMessage{
		Message: ffMsg,
	}

	if p.signer != nil {
		var err error
		signedFFMsg, err = createSignedOrderedMessage(&factorfightpb.SignedFactorFightMessage{}, p.signer, ffMsg, nil, 1)
		if err != nil {
			return err
		}
	}

	return p.ffStream.Send(signedFFMsg)
}

// Recv receives a message from the stream.
func (p dicePeer) Recv(ctx context.Context) (dice.Message, error) {
	if p.diceStream != nil {
		msg, err := p.diceStream.Recv()
		if err != nil {
			return dice.Message{}, err
		}

		// Verify signature if verifier exists
		if p.verifier != nil {
			if msg.Signature == nil {
				return dice.Message{}, fmt.Errorf("message is not signed but verifier is configured")
			}

			msgBytes, err := serializeMessage(msg.Message)
			if err != nil {
				return dice.Message{}, fmt.Errorf("failed to serialize message: %w", err)
			}

			if err := p.verifier.VerifyMessageSignature(msgBytes, msg.Signature); err != nil {
				return dice.Message{}, fmt.Errorf("signature verification failed: %w", err)
			}
		}

		return convertdicepbMessageToInternal(msg.Message), nil
	}

	msg, err := p.ffStream.Recv()
	if err != nil {
		return dice.Message{}, err
	}

	// Verify signature if signedServer exists
	if p.verifier != nil {
		if msg.Signature == nil {
			return dice.Message{}, fmt.Errorf("message is not signed but verifier is configured")
		}

		msgBytes, err := serializeMessage(msg.Message)
		if err != nil {
			return dice.Message{}, fmt.Errorf("failed to serialize message: %w", err)
		}

		if err := p.verifier.VerifyOrderedSignature(msgBytes, msg.Signature); err != nil {
			return dice.Message{}, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	switch m := msg.Message.Message.(type) {
	case *factorfightpb.FactorFightMessage_DiceMsg:
		return convertdicepbMessageToInternal(m.DiceMsg), nil
	default:
		return dice.Message{}, fmt.Errorf("expected dice message, got different type")
	}
}

// diceService implements the gRPC server for Dice.
type diceService struct {
	dicepb.UnimplementedDiceServiceServer
	keyManager *identity.KeyManager
	serverAddr string
	signedMode bool
}

// StreamGame handles the gRPC streaming communication.
func (s *diceService) Roll(stream dicepb.DiceService_RollServer) error {
	signer := NewSigner(s.keyManager, s.serverAddr)
	verifier := NewVerifier(5 * time.Minute)
	dicePeer := dicePeer{
		diceStream: stream,
		signer:     signer,
		verifier:   verifier,
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

func playHighRoll(ctx context.Context, conn *grpc.ClientConn, keyManager *identity.KeyManager, serverAddr string) (*gamepb.ChallengeResponse, error) {
	client := dicepb.NewDiceServiceClient(conn)

	stream, err := client.Roll(ctx)
	if err != nil {
		return nil, err
	}

	signer := NewSigner(keyManager, serverAddr)
	verifier := NewVerifier(5 * time.Minute)
	dicePeer := dicePeer{
		diceStream: stream,
		signer:     signer,
		verifier:   verifier,
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
