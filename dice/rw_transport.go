package dice

import (
	"context"
	"io"
)

// RWTransport is a simple Peer implementation that uses a Reader/Writer to transport data
type RWTransport struct {
	in  chan [32]byte
	out chan [32]byte
}

func NewRWTransport(r io.Reader, w io.Writer) RWTransport {
	t := RWTransport{
		in:  make(chan [32]byte),
		out: make(chan [32]byte),
	}

	// Reader loop
	go func() {
		defer close(t.in)
		for {
			msg, err := readMessage(r)
			if err != nil {
				return
			}
			t.in <- msg
		}
	}()

	// Writer loop
	go func() {
		for msg := range t.out {
			_ = writeMessage(w, msg)
		}
	}()

	return t
}

func (t RWTransport) Send(ctx context.Context, msg [32]byte) error {
	select {
	case t.out <- msg:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (t RWTransport) Recv(ctx context.Context) ([32]byte, error) {
	select {
	case result := <-t.in:
		return result, nil
	case <-ctx.Done():
		return [32]byte{}, ctx.Err()
	}
}

func writeMessage(w io.Writer, msg [32]byte) error {
	_, err := w.Write(msg[:])
	return err
}

func readMessage(r io.Reader) ([32]byte, error) {
	var msg [32]byte

	if _, err := io.ReadFull(r, msg[:]); err != nil {
		return msg, err
	}

	return msg, nil
}
