package identity

import (
	"crypto/sha256"
	"fmt"
)

type HashChain struct {
	previousHash []byte
	sequence     uint64
}

func NewHashChain() *HashChain {
	return &HashChain{
		previousHash: make([]byte, 0),
		sequence:     0,
	}
}

func (hc *HashChain) NextHash(data []byte) []byte {
	hashInput := make([]byte, 0, len(hc.previousHash)+len(data)+8)
	hashInput = append(hashInput, hc.previousHash...)
	hashInput = append(hashInput, data...)

	sequenceBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		sequenceBytes[i] = byte(hc.sequence >> (i * 8))
	}
	hashInput = append(hashInput, sequenceBytes...)

	hash := sha256.Sum256(hashInput)

	hc.previousHash = hash[:]
	hc.sequence++

	return hash[:]
}

func (hc *HashChain) CurrentHash() []byte {
	return hc.previousHash
}

func (hc *HashChain) Sequence() uint64 {
	return hc.sequence
}

func (hc *HashChain) VerifyHash(expectedHash []byte, data []byte, sequence uint64) bool {
	if sequence != hc.sequence {
		return false
	}

	hashInput := make([]byte, 0, len(hc.previousHash)+len(data)+8)
	hashInput = append(hashInput, hc.previousHash...)
	hashInput = append(hashInput, data...)

	sequenceBytes := make([]byte, 8)
	for i := range 8 {
		sequenceBytes[i] = byte(sequence >> (i * 8))
	}
	hashInput = append(hashInput, sequenceBytes...)

	hash := sha256.Sum256(hashInput)

	return fmt.Sprintf("%x", hash) == fmt.Sprintf("%x", expectedHash)
}
