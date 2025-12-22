package dice

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"syscall"
	"time"
)

const (
	retries    = 5
	retryDelay = 500 * time.Millisecond
)

// HTTPTransport implements a Peer that works over an HTTP connection
type HTTPTransport struct {
	sendAddr string
	name     string

	in chan [32]byte
}

func NewHTTPTransport(sendAddr string) *HTTPTransport {
	tport := &HTTPTransport{
		sendAddr: sendAddr,
		in:       make(chan [32]byte, 1),
	}

	mux := http.NewServeMux()
	mux.Handle("POST /recv", tport)

	return tport
}

func (t *HTTPTransport) SetSendAddr(sendAddr string) {
	t.sendAddr = sendAddr
}

func (t *HTTPTransport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var msg [32]byte
	if _, err := io.ReadFull(r.Body, msg[:]); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	t.in <- msg
	w.WriteHeader(http.StatusOK)
}

func (t *HTTPTransport) Send(ctx context.Context, b [32]byte) error {
	for range retries {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.sendAddr+"/recv", bytes.NewReader(b[:]))
		if err != nil {
			return err
		}
		req.Header.Add("Content-Type", "application/octet-stream")

		resp, err := http.DefaultClient.Do(req)
		// retry connection errors
		if errors.Is(err, syscall.ECONNREFUSED) {
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}

	return nil
}

func (t *HTTPTransport) Recv(ctx context.Context) ([32]byte, error) {
	select {
	case result := <-t.in:
		return result, nil
	case <-ctx.Done():
		return [32]byte{}, ctx.Err()
	}
}
