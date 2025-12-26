package dice

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	commitLabel = "dice-commit-v1|"
	rollLabel   = "dice-roll-v1|"

	max = uint16(^uint16(0)) // 2^16 - 1
)

// Peer is used to send and receive messages and can be used to implement the
// Roller over the network or with other communication methods
type Peer interface {
	// Send is used to send data to the Peer. It should not block while waiting for the Peer to receive.
	Send(context.Context, Message) error
	// Recv is used to receive data from the other Peer. It should block until data is received
	Recv(context.Context) (Message, error)
}

// Roller allows rolling a dice with a peer
type Roller struct {
	name  string
	sides uint8

	rollValueLimit uint16

	in   chan [32]byte
	peer Peer
}

// NewRoller creates a Roller with the specified name and number of sides.
func NewRoller(name string, sides uint8, peer Peer) (Roller, error) {
	// this limit can be increased
	if sides > 20 {
		return Roller{}, errors.New("max size of die is 20")
	}
	return Roller{
		name:           name,
		sides:          sides,
		rollValueLimit: max - (max % uint16(sides)),
		in:             make(chan [32]byte, 1),
		peer:           peer,
	}, nil
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
	result chan []uint16
	err    chan error
}

// GetOne gets the first/only dice roll result and error
func (r Roll) GetOne() (uint16, error) {
	rolls, err := r.Get(1)
	if err != nil || len(rolls) == 0 {
		return 0, err
	}
	return rolls[0], nil
}

// Get gets the dice roll result and error
func (r Roll) Get(n uint8) ([]uint16, error) {
	if n <= 0 || n > 8 {
		return nil, errors.New("number of rolls must be >0 and <=8")
	}

	var result []uint16
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

// RollSync will start and complete the roll in one call. It will block to communicate with the peer
func (d Roller) RollSync(ctx context.Context, n uint8) ([]uint16, error) {
	r := d.Roll(ctx)
	return r.Get(n)
}

// Roll initializes the roll process with the peer and returns the RollN to receive asynchronous results
// It rolls up to 8 dice simultaneously
func (d Roller) Roll(ctx context.Context) Roll {
	roll := Roll{
		result: make(chan []uint16),
		err:    make(chan error),
	}

	secret, commitment := createSecretAndCommitment()

	// finish is used to consistently push results to both channels which is required to finish the roll
	finish := func(v []uint16, err error) {
		roll.result <- v
		roll.err <- err
	}

	go func() {
		exchange := d.startExchange(ctx, d.peer, secret, commitment)

		peerSecret, err := exchange(ctx)
		if err != nil {
			finish(nil, fmt.Errorf("failed to exchange: %w", err))
			return
		}

		secretA, secretB := secret[:], peerSecret[:]
		if bytes.Compare(secretA, secretB) > 0 {
			secretA, secretB = secretB, secretA
		}
		concatSecrets := append(secretA, secretB...)

		sum := sha256.Sum256(append([]byte(rollLabel), concatSecrets...))

		rolls := d.roll(sum)

		err = d.confirmRoll(ctx, rolls)
		if err != nil {
			finish(nil, err)
			return
		}

		finish(rolls, nil)
	}()

	return roll
}

func (d *Roller) roll(sum [32]byte) []uint16 {
	// always "roll" 8 dice so the result is flexible
	const n = 8

	result := make([]uint16, n)

	var written uint8 = 0

	for i := 0; i+2 <= len(sum) && written < n; i += 2 {
		v := binary.BigEndian.Uint16(sum[i : i+2])

		if v < d.rollValueLimit {
			result[written] = v%uint16(d.sides) + 1
			written++
		}
	}

	if written != n {
		panic("not enough entropy to produce all dice rolls")
	}

	return result
}

func (d *Roller) confirmRoll(ctx context.Context, rolls []uint16) error {
	var rollValueOut [32]byte
	n := 0
	for i := range len(rolls) {
		binary.BigEndian.PutUint16(rollValueOut[n:n+2], rolls[i])
		n += 2
	}

	err := d.peer.Send(ctx, Message{MsgTypeRollConfirmation, rollValueOut})
	if err != nil {
		return fmt.Errorf("failed to send roll confirmation: %w", err)
	}

	msg, err := d.peer.Recv(ctx)
	if err != nil {
		return fmt.Errorf("failed to receive roll confirmation: %w", err)
	}
	if msg.Type != MsgTypeRollConfirmation {
		return fmt.Errorf("received unexpected MsgType for roll confirmation: %s", msg.Type.String())
	}

	if rollValueOut != msg.Data {
		return fmt.Errorf("error confirming roll: peer (%s) != self (%s)", hex.EncodeToString(msg.Data[:]), hex.EncodeToString(rollValueOut[:]))
	}

	return nil
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
		err := peer.Send(ctx, Message{MsgTypeCommitment, commitment})
		if err != nil {
			errCh <- fmt.Errorf("failed to send secret: %w", err)
			return
		}

		// 2. Receive commitment
		commitmentMsg, err := peer.Recv(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to receive commitment: %w", err)
			return
		}
		if commitmentMsg.Type != MsgTypeCommitment {
			errCh <- fmt.Errorf("received unexpected MsgType for commitment: %s", commitmentMsg.Type.String())
			return
		}

		// 3. Send secret
		err = peer.Send(ctx, Message{MsgTypeSecret, secret})
		if err != nil {
			errCh <- fmt.Errorf("failed to send secret: %w", err)
			return
		}

		// 4. Receive secret
		secretMsg, err := peer.Recv(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to receive secret: %w", err)
			return
		}
		if secretMsg.Type != MsgTypeSecret {
			errCh <- fmt.Errorf("received unexpected MsgType for secret: %s", secretMsg.Type.String())
			return
		}

		// 5. Complete by sending data to channels and closing
		select {
		case secretCh <- secretMsg.Data:
			close(secretCh)
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		}
		select {
		case commitCh <- commitmentMsg.Data:
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
