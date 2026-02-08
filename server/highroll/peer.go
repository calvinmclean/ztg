package highroll

import (
	"context"
	"fmt"

	"github.com/calvinmclean/ztg/dice"
	dicepb "github.com/calvinmclean/ztg/gen/go/proto/dice/v1"
	"github.com/calvinmclean/ztg/server"
)

type stream interface {
	Recv() (*dicepb.SignedMessage, error)
	Send(*dicepb.SignedMessage) error
}

type peer struct {
	highrollStream stream
	signer         *server.Signer
	verifier       *server.Verifier
	lastHash       []byte
	sequence       uint64
}

var _ dice.Peer = (*peer)(nil)

func (p *peer) Send(ctx context.Context, msg dice.Message) error {
	protoMsg := convertInternalDiceMessageToProto(msg)
	signedMsg, err := server.CreateSignedMessage(&dicepb.SignedMessage{}, p.signer, protoMsg)
	if err != nil {
		return err
	}
	return p.highrollStream.Send(signedMsg)
}

func (p *peer) Recv(ctx context.Context) (dice.Message, error) {
	msg, err := p.highrollStream.Recv()
	if err != nil {
		return dice.Message{}, err
	}
	if err := p.verifier.VerifySignatureProto(msg.Message, msg.Signature); err != nil {
		return dice.Message{}, fmt.Errorf("signature verification failed: %w", err)
	}
	return convertdicepbMessageToInternal(msg.Message), nil
}

func newPeer(highrollStream stream, signer *server.Signer, verifier *server.Verifier) *peer {
	return &peer{
		highrollStream: highrollStream,
		signer:         signer,
		verifier:       verifier,
	}
}
