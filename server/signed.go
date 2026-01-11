package server

import (
	"fmt"

	dicepb "ztg/gen/go/dice/v1"
	factorfightpb "ztg/gen/go/factorfight/v1"
	identitypb "ztg/gen/go/identity/v1"
	"ztg/identity"

	"google.golang.org/protobuf/proto"
)

type SignedServer struct {
	keyManager *identity.KeyManager
	signer     *identity.Signer
}

func NewSignedServer(keyManager *identity.KeyManager, serverAddr string) *SignedServer {
	return &SignedServer{
		keyManager: keyManager,
		signer:     identity.NewSigner(keyManager.PrivateKey(), serverAddr),
	}
}

func (s *SignedServer) verifyMessageSignature(message []byte, signature *identitypb.Signature) error {
	if signature == nil {
		return fmt.Errorf("message is not signed")
	}

	if len(signature.Signature) == 0 {
		return fmt.Errorf("empty signature")
	}

	if signature.SignerAddress == "" {
		return fmt.Errorf("empty signer address")
	}

	// For verification, we need to create a signer with the appropriate private key
	// In a real implementation, you'd look up the public key by address from a registry
	// For now, we'll use our own private key to verify (this assumes we're verifying our own messages)
	signer := identity.NewSigner(s.keyManager.PrivateKey(), signature.SignerAddress)

	// Verify the signature
	err := signer.Verify(message, signature.Signature, signature.SignerAddress)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

func (s *SignedServer) verifyOrderedSignature(message []byte, signature *identitypb.OrderedSignature) error {
	if signature == nil {
		return fmt.Errorf("message is not signed")
	}

	if len(signature.Signature) == 0 {
		return fmt.Errorf("empty signature")
	}

	if signature.SignerAddress == "" {
		return fmt.Errorf("empty signer address")
	}

	if signature.Sequence == 0 {
		return fmt.Errorf("sequence must be greater than 0")
	}

	// Create a signer for verification
	signer := identity.NewSigner(s.keyManager.PrivateKey(), signature.SignerAddress)

	// Verify the signature
	err := signer.Verify(message, signature.Signature, signature.SignerAddress)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// TODO: implement hash chain verification for PreviousHash
	// For now, we'll just accept any previous hash

	return nil
}

func (s *SignedServer) signMessage(message []byte) *identitypb.Signature {
	signature, err := s.signer.Sign(message)
	if err != nil {
		return nil
	}

	return &identitypb.Signature{
		Signature:     signature,
		SignerAddress: s.signer.Address(),
	}
}

func (s *SignedServer) signOrderedMessage(message []byte, previousHash []byte, sequence uint64) *identitypb.OrderedSignature {
	hashChain := &identity.HashChain{}
	hashChain.NextHash(message)

	signature, err := s.signer.Sign(message)
	if err != nil {
		return nil
	}

	return &identitypb.OrderedSignature{
		Signature:     signature,
		SignerAddress: s.signer.Address(),
		PreviousHash:  previousHash,
		Sequence:      sequence,
	}
}

func (s *SignedServer) serializeMessage(msg proto.Message) ([]byte, error) {
	return proto.Marshal(msg)
}

func (s *SignedServer) createSignedDiceMessage(msg *dicepb.Message) (*dicepb.SignedMessage, error) {
	messageBytes, err := s.serializeMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	signature := s.signMessage(messageBytes)

	return &dicepb.SignedMessage{
		Message:   msg,
		Signature: signature,
	}, nil
}

func (s *SignedServer) createSignedFactorFightMessage(msg *factorfightpb.FactorFightMessage, previousHash []byte, sequence uint64) (*factorfightpb.SignedFactorFightMessage, error) {
	messageBytes, err := s.serializeMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	signature := s.signOrderedMessage(messageBytes, previousHash, sequence)

	return &factorfightpb.SignedFactorFightMessage{
		Message:   msg,
		Signature: signature,
	}, nil
}
