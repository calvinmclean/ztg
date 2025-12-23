package dice

import (
	"context"
	"io"
)

// PipePeer is a simple Peer implementation that uses a Reader/Writer to transport data
type PipePeer struct {
	out chan Message

	r io.Reader
	w io.Writer
}

var _ Peer = PipePeer{}

// NewPipePeers creates two Peers that use io.Pipe to communicate. This creates two
// instead of one because it only works if the Peers share the Pipe in the correct way and
// also only works when running the Peers in the same program. It relies on io.Pipe being synchronous
func NewPipePeers() (PipePeer, PipePeer) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	t1 := PipePeer{
		out: make(chan Message),
		r:   r1,
		w:   w2,
	}

	t2 := PipePeer{
		out: make(chan Message),
		r:   r2,
		w:   w1,
	}

	// Since Write blocks on a PipeWriter, this needs to be done async
	// to prevent Send from blocking
	go func() {
		for msg := range t1.out {
			_, _ = t1.w.Write(msg.Bytes())
		}
	}()
	go func() {
		for msg := range t2.out {
			_, _ = t2.w.Write(msg.Bytes())
		}
	}()

	return t1, t2
}

func (t PipePeer) Send(ctx context.Context, msg Message) error {
	select {
	case t.out <- msg:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (t PipePeer) Recv(ctx context.Context) (Message, error) {
	msg, err := ReadMessage(t.r)
	if err != nil {
		return Message{}, err
	}

	return msg, nil
}
