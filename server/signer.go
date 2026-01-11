package server

import (
	"fmt"
	dicepb "ztg/gen/go/dice/v1"
	factorfightpb "ztg/gen/go/factorfight/v1"
	identitypb "ztg/gen/go/identity/v1"
	"ztg/identity"

	"google.golang.org/protobuf/proto"
)

// Signer handles cryptographic signing operations
type Signer struct {
	signer *identity.Signer
}

// NewSigner creates a new Signer instance
func NewSigner(keyManager *identity.KeyManager, serverAddr string) *Signer {
	return &Signer{
		signer: identity.NewSigner(keyManager.PrivateKey(), serverAddr),
	}
}

// SignMessage signs a message and returns a signature
func (s *Signer) SignMessage(message []byte) (*identitypb.Signature, error) {
	signature, err := s.signer.Sign(message)
	if err != nil {
		return nil, err
	}

	return &identitypb.Signature{
		Signature:     signature,
		SignerAddress: s.signer.Address(),
	}, nil
}

// SignOrderedMessage signs a message with sequence and previous hash
func (s *Signer) SignOrderedMessage(message []byte, previousHash []byte, sequence uint64) (*identitypb.OrderedSignature, error) {
	signature, err := s.signer.Sign(message)
	if err != nil {
		return nil, err
	}

	return &identitypb.OrderedSignature{
		Signature:     signature,
		SignerAddress: s.signer.Address(),
		PreviousHash:  previousHash,
		Sequence:      sequence,
	}, nil
}

// Address returns the signer's address
func (s *Signer) Address() string {
	return s.signer.Address()
}

// serializeMessage is a utility function to marshal protobuf messages
func serializeMessage(msg proto.Message) ([]byte, error) {
	return proto.Marshal(msg)
}

// createSignedDiceMessage creates a signed dice message using a Signer
func createSignedDiceMessage(signer *Signer, msg *dicepb.Message) (*dicepb.SignedMessage, error) {
	messageBytes, err := serializeMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	signature, err := signer.SignMessage(messageBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return &dicepb.SignedMessage{
		Message:   msg,
		Signature: signature,
	}, nil
}

// createSignedFactorFightMessage creates a signed factorfight message using a Signer
func createSignedFactorFightMessage(signer *Signer, msg *factorfightpb.FactorFightMessage, previousHash []byte, sequence uint64) (*factorfightpb.SignedFactorFightMessage, error) {
	messageBytes, err := serializeMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	signature, err := signer.SignOrderedMessage(messageBytes, previousHash, sequence)
	if err != nil {
		return nil, fmt.Errorf("failed to sign ordered message: %w", err)
	}

	return &factorfightpb.SignedFactorFightMessage{
		Message:   msg,
		Signature: signature,
	}, nil
}
