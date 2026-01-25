package factorfight

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/calvinmclean/ztg/dice"
	"github.com/calvinmclean/ztg/factorfight"
	factorfightpb "github.com/calvinmclean/ztg/gen/go/factorfight/v1"
	"github.com/calvinmclean/ztg/server"

	"google.golang.org/protobuf/proto"
)

// peer implements the Peer interface for use with streaming.
type peer struct {
	stream   stream
	signer   *server.Signer
	verifier *server.Verifier

	// Hash chain tracking for ordered signatures
	lastHash []byte
	sequence uint64
}

var _ factorfight.Peer = (*peer)(nil)

func (p *peer) Dice() dice.Peer {
	// factorfightPeer also implements dice.Peer
	return p
}

// SendMove sends a move to the stream.
func (p *peer) SendMove(ctx context.Context, move factorfight.Move) error {
	protoMove := convertInternalFactorfightMoveToProto(move)
	ffMsg := &factorfightpb.FactorFightMessage{
		Message: &factorfightpb.FactorFightMessage_Move{Move: protoMove},
	}

	// Calculate hash of this message for hash chain
	msgBytes, err := proto.Marshal(ffMsg)
	if err != nil {
		return fmt.Errorf("failed to serialize message for hash chain: %w", err)
	}

	hash := sha256.Sum256(msgBytes)

	// Increment sequence for ordered signature
	p.sequence++

	signedMsg, err := server.CreateSignedOrderedMessage(&factorfightpb.SignedFactorFightMessage{}, p.signer, ffMsg, p.lastHash, p.sequence)
	if err != nil {
		return err
	}

	// Update last hash for next message
	p.lastHash = hash[:]

	return p.stream.Send(signedMsg)
}

type stream interface {
	Recv() (*factorfightpb.SignedFactorFightMessage, error)
	Send(*factorfightpb.SignedFactorFightMessage) error
}

// RecvMove receives a move from the stream.
func (p *peer) RecvMove(ctx context.Context) (factorfight.Move, error) {
	msg, err := p.stream.Recv()
	if err != nil {
		return factorfight.Move{}, err
	}

	if err := p.verifier.VerifyOrderedSignatureProto(msg.Message, msg.Signature); err != nil {
		return factorfight.Move{}, fmt.Errorf("signature verification failed: %w", err)
	}

	switch m := msg.Message.Message.(type) {
	case *factorfightpb.FactorFightMessage_Move:
		return convertfactorfightpbMoveToInternal(m.Move), nil
	default:
		return factorfight.Move{}, fmt.Errorf("expected move, got different message type")
	}
}

// Send sends a message to the stream.
func (p *peer) Send(ctx context.Context, msg dice.Message) error {
	protoMsg := convertInternalDiceMessageToProto(msg)
	ffMsg := &factorfightpb.FactorFightMessage{
		Message: &factorfightpb.FactorFightMessage_DiceMsg{DiceMsg: protoMsg},
	}

	// Calculate hash of this message for hash chain
	msgBytes, err := proto.Marshal(ffMsg)
	if err != nil {
		return fmt.Errorf("failed to serialize message for hash chain: %w", err)
	}

	hash := sha256.Sum256(msgBytes)

	// Increment sequence for ordered signature
	p.sequence++

	signedFFMsg, err := server.CreateSignedOrderedMessage(&factorfightpb.SignedFactorFightMessage{}, p.signer, ffMsg, p.lastHash, p.sequence)
	if err != nil {
		return err
	}

	// Update last hash for next message
	p.lastHash = hash[:]

	return p.stream.Send(signedFFMsg)
}

// Recv receives a message from the stream.
func (p *peer) Recv(ctx context.Context) (dice.Message, error) {
	msg, err := p.stream.Recv()
	if err != nil {
		return dice.Message{}, err
	}

	if err := p.verifier.VerifyOrderedSignatureProto(msg.Message, msg.Signature); err != nil {
		return dice.Message{}, fmt.Errorf("signature verification failed: %w", err)
	}

	switch m := msg.Message.Message.(type) {
	case *factorfightpb.FactorFightMessage_DiceMsg:
		return convertdicepbMessageToInternal(m.DiceMsg), nil
	default:
		return dice.Message{}, fmt.Errorf("expected dice message, got different type")
	}
}

func createFactorfightPeer(stream stream, signer *server.Signer, verifier *server.Verifier) *peer {
	return &peer{
		stream:   stream,
		signer:   signer,
		verifier: verifier,
	}
}
