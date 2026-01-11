package server

import (
	"ztg/dice"
	"ztg/factorfight"
	"ztg/identity"
)

// createSignerVerifierPair creates a signer and verifier pair if signed mode is enabled
func createSignerVerifierPair(keyManager *identity.KeyManager, serverAddr string, signedMode bool) (*Signer, *Verifier) {
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

// createDicePeerForFactorFight creates a dicePeer specifically for FactorFight games
func createDicePeerForFactorFight(ffStream factorfightStream, signer *Signer, verifier *Verifier) *dicePeer {
	return &dicePeer{
		ffStream: ffStream,
		signer:   signer,
		verifier: verifier,
	}
}

// createFactorfightPeer creates a factorfightPeer with the specified parameters
func createFactorfightPeer(stream factorfightStream, dicePeer *dicePeer, signer *Signer, verifier *Verifier) *factorfightPeer {
	return &factorfightPeer{
		stream:   stream,
		dicePeer: dicePeer,
		signer:   signer,
		verifier: verifier,
	}
}

// resolveStrategy returns the provided strategy or the default strategy if nil
func resolveStrategy(strategy factorfight.Strategy) factorfight.Strategy {
	if strategy != nil {
		return strategy
	}
	return factorfight.DefaultStrategy
}

// createRoller creates a dice roller with the specified peer
func createRoller(peer dice.Peer, sides int) (dice.Roller, error) {
	if sides <= 0 {
		sides = DefaultDieSides
	}
	return dice.NewRoller(uint8(sides), peer)
}
