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

// Peer is used to send and receive messages and can be used to implement the
// Roller over the network or with other communication methods
type Peer interface {
	Send([32]byte)
	Recv() <-chan [32]byte
}

// Roller allows rolling a dice with a peer
type Roller struct {
	name  string
	sides uint64

	in   chan [32]byte
	peer Peer
}

// NewRoller creates a Roller with the specified name and number of sides.
func NewRoller(name string, sides uint8, peer Peer) Roller {
	return Roller{
		name:  name,
		sides: uint64(sides),
		in:    make(chan [32]byte, 1),
		peer:  peer,
	}
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
	err    chan error
}

// Result gets the dice roll result and error
func (r Roll) Result() (int, error) {
	return <-r.result, <-r.err
}

// Roll initializes the roll process with the peer and returns the Roll to receive asynchronous results
func (d Roller) Roll() Roll {
	roll := Roll{
		result: make(chan int),
		err:    make(chan error),
	}

	secret, commitment := createSecretAndCommitment()

	go func() {
		exchange := d.startExchange(d.peer, secret, commitment)

		peerSecret, err := exchange()
		if err != nil {
			roll.result <- 0
			roll.err <- err
			return
		}

		secretA, secretB := secret[:], peerSecret[:]
		if bytes.Compare(secretA, secretB) > 0 {
			secretA, secretB = secretB, secretA
		}
		concatSecrets := append(secretA, secretB...)

		sum := sha256.Sum256(append([]byte(rollLabel), concatSecrets...))

		// Use rejection sampling to avoid modulo bias
		// Take 64 bits at a time
		const max = uint64(^uint64(0)) // 2^64 - 1
		limit := max - (max % d.sides)

		for i := 0; i+8 <= len(sum); i += 8 {
			v := binary.BigEndian.Uint64(sum[i : i+8])
			if v < limit {
				roll.result <- int(v%d.sides) + 1
				roll.err <- nil
				return
			}
		}

		// TODO: send roll result to peer to confirm and finalize? Both peers should confirm the same result before it is considered valid
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
func (d *Roller) startExchange(peer Peer, secret, commitment [32]byte) func() ([32]byte, error) {
	secretCh := make(chan [32]byte)
	commitCh := make(chan [32]byte)

	go func() {
		// 1. Send commitment
		peer.Send(commitment)

		// 2. Receive commitment
		otherCommit := <-peer.Recv()

		// 3. Send secret
		peer.Send(secret)

		// 4. Receive secret
		otherSecret := <-peer.Recv()

		// 5. Complete by sending data to channels and closing
		secretCh <- otherSecret
		commitCh <- otherCommit
		close(secretCh)
		close(commitCh)
	}()

	// this function will block until the above goroutine finishes by pushing secret and commitment into the
	// channels
	return func() ([32]byte, error) {
		secret := <-secretCh
		commit := <-commitCh
		return secret, verifyCommitment(secret, commit)
	}
}
