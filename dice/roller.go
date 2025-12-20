package dice

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

const (
	commitLabel = "dice-commit-v1|"
	rollLabel   = "dice-roll-v1|"
)

// Peer abstracts the Roller and will allow testing how we handle a malicious Peer
type Peer interface {
	Roll(Peer) Roll
	receiver() chan<- [32]byte
}

// Roller allows rolling a dice with a peer
type Roller struct {
	name string

	secret     [32]byte
	commitment [32]byte

	in chan [32]byte
}

// NewRoller creates a Roller with the specified name and a randomly-generated secret and commitment hash
func NewRoller(name string) Roller {
	secret, commitment := createSecretAndCommitment()
	return Roller{
		name:       name,
		secret:     secret,
		commitment: commitment,
		in:         make(chan [32]byte, 1),
	}
}

// receiver returns the channel for receiving data from a Peer
func (d Roller) receiver() chan<- [32]byte {
	return d.in
}

// createSecretAndCommitment generates a random secret and calculates the commitment hash
func createSecretAndCommitment() ([32]byte, [32]byte) {
	var secret [32]byte
	_, _ = rand.Read(secret[:])

	commitment := calculateCommitment(secret)

	return secret, commitment
}

// calculateCommitment creates a SHA256 sum of the secret prepended with the label
func calculateCommitment(secret [32]byte) [32]byte {
	commitment := sha256.Sum256(append([]byte(commitLabel), secret[:]...))
	return commitment
}

// verifyCommitment hashes the secret and compares to the expected commitment
func verifyCommitment(secret, expectedCommit [32]byte) error {
	calculatedCommit := calculateCommitment(secret)
	if calculatedCommit != expectedCommit {
		return errors.New("failed to verify roll with peer")
	}
	return nil
}

// Roll allows asynchronous retrieval of the dice roll result or the error
type Roll struct {
	result chan int
	err    error
}

// Result gets the dice toll result and error
func (r Roll) Result() (int, error) {
	return <-r.result, r.err
}

// Roll initializes the roll process with the peer and returns the Roll to receive asynchronous results
func (d Roller) Roll(peer Peer) Roll {
	roll := Roll{
		result: make(chan int),
	}

	go func() {
		exchange := d.startExchange(peer)

		peerSecret, err := exchange()
		if err != nil {
			roll.err = err
			roll.result <- 0
			return
		}

		secretA, secretB := d.secret[:], peerSecret[:]
		if bytes.Compare(secretA, secretB) > 0 {
			secretA, secretB = secretB, secretA
		}
		concatSecrets := append(secretA, secretB...)

		sum := sha256.Sum256(append([]byte(rollLabel), concatSecrets...))

		// Use rejection sampling to avoid modulo bias
		// Take 64 bits at a time
		const sides = 6
		const max = uint64(^uint64(0)) // 2^64 - 1
		const limit = max - (max % sides)

		for i := 0; i+8 <= len(sum); i += 8 {
			v := binary.BigEndian.Uint64(sum[i : i+8])
			if v < limit {
				roll.result <- int(v%uint64(sides)) + 1
				return
			}
		}
	}()

	return roll
}

// startExchange initializes the dice exchange with the Peer. This starts a goroutine and
// returns a blocking function to receive the results. The goroutine is used because the peer
// will be running the same goroutine to perform the exchange in this order:
//  1. send commitment to the Peer
//  2. receive the Peer's commitment
//  3. send secret to the Peer
//  4. receive the secret from the Peer
//  5. Send the Secret and Commitment to output channels
func (d *Roller) startExchange(peer Peer) func() ([32]byte, error) {
	exchangeSecret := make(chan [32]byte)
	exchangeCommitment := make(chan [32]byte)
	go func() {
		out := peer.receiver()

		out <- d.commitment

		otherCommit := <-d.in

		// Now send the secret. Since it is using the buffered channel, it will not send until the commitment is received
		// Also, it runs after we receive the peer's commitment since the receive above will block
		out <- d.secret

		exchangeSecret <- <-d.in
		close(exchangeSecret)
		exchangeCommitment <- otherCommit
		close(exchangeCommitment)
	}()

	// this function will block until the above goroutine finishes by pushing secret and commitment into the
	// channels
	return func() (b [32]byte, err error) {
		secret := <-exchangeSecret
		commitment := <-exchangeCommitment
		return secret, verifyCommitment(secret, commitment)
	}
}
