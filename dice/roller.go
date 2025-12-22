package dice

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	commitLabel = "dice-commit-v1|"
	rollLabel   = "dice-roll-v1|"
)

// Peer is used to send and receive messages and can be used to implement the
// Roller over the network or with other communication methods
type Peer interface {
	// Send is used to send data to the Peer. It should not block while waiting for the Peer to receive.
	Send(context.Context, [32]byte) error
	// Recv is used to receive data from the other Peer. It should block until data is received
	Recv(context.Context) ([32]byte, error)
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
	var result int
	var err error

	// receive values in either order
	select {
	case result = <-r.result:
		err = <-r.err
	case err = <-r.err:
		result = <-r.result
	}
	return result, err
}

// Roll initializes the roll process with the peer and returns the Roll to receive asynchronous results
func (d Roller) Roll(ctx context.Context) Roll {
	roll := Roll{
		result: make(chan int),
		err:    make(chan error),
	}

	secret, commitment := createSecretAndCommitment()

	// finish is used to consistently push results to both channels which is required to finish the roll
	finish := func(v int, err error) {
		roll.result <- v
		roll.err <- err
	}

	go func() {
		exchange := d.startExchange(ctx, d.peer, secret, commitment)

		peerSecret, err := exchange(ctx)
		if err != nil {
			finish(0, err)
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

		var rollValue uint64
		for i := 0; i+8 <= len(sum); i += 8 {
			v := binary.BigEndian.Uint64(sum[i : i+8])
			if v < limit {
				rollValue = v%d.sides + 1
				break
			}
		}

		var rollValueOut [32]byte
		binary.BigEndian.PutUint64(rollValueOut[:], rollValue)

		err = d.peer.Send(ctx, rollValueOut)
		if err != nil {
			finish(0, err)
			return
		}

		confirmation, err := d.peer.Recv(ctx)
		if err != nil {
			finish(0, err)
			return
		}

		confirmationVal := binary.BigEndian.Uint64(confirmation[:])

		if rollValue != confirmationVal {
			finish(0, fmt.Errorf("error confirming roll: peer (%d) != self (%d)", confirmationVal, rollValue))
			return
		}

		finish(int(rollValue), nil)
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
func (d *Roller) startExchange(ctx context.Context, peer Peer, secret, commitment [32]byte) func(ctx context.Context) ([32]byte, error) {
	secretCh := make(chan [32]byte)
	commitCh := make(chan [32]byte)
	errCh := make(chan error)

	go func() {
		// 1. Send commitment
		err := peer.Send(ctx, commitment)
		if err != nil {
			errCh <- fmt.Errorf("failed to send secret: %w", err)
			return
		}

		// 2. Receive commitment
		otherCommit, err := peer.Recv(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to receive commitment: %w", err)
			return
		}

		// 3. Send secret
		err = peer.Send(ctx, secret)
		if err != nil {
			errCh <- fmt.Errorf("failed to send secret: %w", err)
			return
		}

		// 4. Receive secret
		otherSecret, err := peer.Recv(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to receive secret: %w", err)
			return
		}

		// 5. Complete by sending data to channels and closing
		select {
		case secretCh <- otherSecret:
			close(secretCh)
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		}
		select {
		case commitCh <- otherCommit:
			close(commitCh)
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		}
	}()

	// this function will block until the above goroutine finishes by pushing secret and commitment into the
	// channels
	return func(ctx context.Context) ([32]byte, error) {
		var secret, commit [32]byte
		select {
		case secret = <-secretCh:
		case err := <-errCh:
			return [32]byte{}, err
		case <-ctx.Done():
			return [32]byte{}, ctx.Err()
		}

		select {
		case commit = <-commitCh:
		case err := <-errCh:
			return [32]byte{}, err
		case <-ctx.Done():
			return [32]byte{}, ctx.Err()
		}
		return secret, verifyCommitment(secret, commit)
	}
}
