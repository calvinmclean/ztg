package server

import (
	"context"
	"fmt"

	"ztg/dice"
	"ztg/identity"

	dicepb "ztg/gen/go/dice/v1"
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

type diceStream interface {
	Recv() (*dicepb.SignedMessage, error)
	Send(*dicepb.SignedMessage) error
}

// dicePeer uses the ffStream OR diceStream to implement a dicePeer. This allows it to be used for FactorFight or just a plain dice game
type dicePeer struct {
	diceStream diceStream
	signer     *Signer
	verifier   *Verifier

	// Hash chain tracking for ordered signatures
	lastHash []byte
	sequence uint64
}

var _ dice.Peer = (*dicePeer)(nil)

// Send sends a message to the stream.
func (p *dicePeer) Send(ctx context.Context, msg dice.Message) error {
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

// Recv receives a message from the stream.
func (p *dicePeer) Recv(ctx context.Context) (dice.Message, error) {
	msg, err := p.diceStream.Recv()
	if err != nil {
		return dice.Message{}, err
	}

	// Verify signature if verifier exists
	// TODO: update to use a nil/mock verifier and abstract this
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

// diceService implements the gRPC server for Dice.
type diceService struct {
	dicepb.UnimplementedDiceServiceServer
	keyManager *identity.KeyManager
	serverAddr string
	signedMode bool
}

// StreamGame handles the gRPC streaming communication.
func (s *diceService) Roll(stream dicepb.DiceService_RollServer) error {
	signer, verifier := createSignerVerifierPair(s.keyManager, s.serverAddr, s.signedMode)

	dicePeer := createDicePeer(stream, signer, verifier)

	roller, err := createRoller(dicePeer, DefaultDieSides)
	if err != nil {
		return err
	}

	_, err = roller.RollSync(stream.Context(), 1)
	if err != nil {
		return err
	}

	return nil
}

func playHighRoll(ctx context.Context, conn *grpc.ClientConn, keyManager *identity.KeyManager, serverAddr string, signedMode bool) (*gamepb.ChallengeResponse, error) {
	client := dicepb.NewDiceServiceClient(conn)

	stream, err := client.Roll(ctx)
	if err != nil {
		return nil, err
	}

	signer, verifier := createSignerVerifierPair(keyManager, serverAddr, signedMode)

	dicePeer := createDicePeer(stream, signer, verifier)

	roller, err := createRoller(dicePeer, DefaultDieSides)
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
