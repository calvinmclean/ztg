package server

import (
	"fmt"

	"ztg/dice"
	identitypb "ztg/gen/go/identity/v1"
	"ztg/identity"

	"google.golang.org/protobuf/proto"
)

// CreateSignerVerifierPair creates a signer and verifier pair if signed mode is enabled
func CreateSignerVerifierPair(keyManager *identity.KeyManager, serverAddr string, signedMode bool) (*Signer, *Verifier) {
	if !signedMode {
		return nil, nil
	}
	return NewSigner(keyManager, serverAddr), NewVerifier(DefaultVerifierTTL)
}

// createDicePeer creates a dicePeer with the specified parameters
func createDicePeer(diceStream diceStream, signer *Signer, verifier *Verifier) *dicePeer {
	return &dicePeer{
		diceStream: diceStream,
		signer:     signer,
		verifier:   verifier,
	}
}

// verifyMessageSignature verifies a regular message signature if a verifier is configured
func verifyMessageSignature(verifier *Verifier, msg proto.Message, signature *identitypb.Signature) error {
	if verifier == nil {
		return nil
	}

	if signature == nil {
		return fmt.Errorf("message is not signed but verifier is configured")
	}

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	return verifier.VerifyMessageSignature(msgBytes, signature)
}

// createRoller creates a dice roller with the specified peer
func createRoller(peer dice.Peer, sides int) (dice.Roller, error) {
	if sides <= 0 {
		sides = DefaultDieSides
	}
	return dice.NewRoller(uint8(sides), peer)
}
