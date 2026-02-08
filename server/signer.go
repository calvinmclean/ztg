package server

import (
	"fmt"

	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"
	"github.com/calvinmclean/ztg/identity"

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

// SignProto marshals and signs a proto message
func (s *Signer) SignProto(msg proto.Message) (*identitypb.Signature, error) {
	messageBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	return s.SignMessage(messageBytes)
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

// SignOrderedMessage signs a message with sequence and previous hash
func (s *Signer) SignOrderedMessageProto(msg proto.Message, previousHash []byte, sequence uint64) (*identitypb.OrderedSignature, error) {
	messageBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	return s.SignOrderedMessage(messageBytes, previousHash, sequence)
}

// Address returns the signer's address
func (s *Signer) Address() string {
	return s.signer.Address()
}

type SignedMessage[T proto.Message] interface {
	SetSignature(*identitypb.Signature)
	SetMessage(T)
}

func CreateSignedMessage[T SignedMessage[R], R proto.Message](result T, signer *Signer, msg R) (T, error) {
	signature, err := signer.SignProto(msg)
	if err != nil {
		return *new(T), fmt.Errorf("failed to sign message: %w", err)
	}

	result.SetMessage(msg)
	result.SetSignature(signature)

	return result, nil
}

type OrderedSignedMessage[T proto.Message] interface {
	SetSignature(*identitypb.OrderedSignature)
	SetMessage(T)
}

func CreateSignedOrderedMessage[T OrderedSignedMessage[R], R proto.Message](result T, signer *Signer, msg R, previousHash []byte, sequence uint64) (T, error) {
	messageBytes, err := proto.Marshal(msg)
	if err != nil {
		return *new(T), fmt.Errorf("failed to serialize message: %w", err)
	}

	signature, err := signer.SignOrderedMessage(messageBytes, previousHash, sequence)
	if err != nil {
		return *new(T), fmt.Errorf("failed to sign ordered message: %w", err)
	}

	result.SetMessage(msg)
	result.SetSignature(signature)

	return result, nil
}
