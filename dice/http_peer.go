package dice

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"syscall"
	"time"
)

const (
	retries    = 5
	retryDelay = 500 * time.Millisecond
)

// HTTPPeer implements a Peer that works over an HTTP connection
type HTTPPeer struct {
	sendAddr string
	name     string

	in chan Message
}

var _ Peer = &HTTPPeer{}

func NewHTTPPeer(addr string) *HTTPPeer {
	tport := &HTTPPeer{
		sendAddr: addr,
		in:       make(chan Message, 1),
	}

	mux := http.NewServeMux()
	mux.Handle("POST /recv", tport)

	return tport
}

func (t *HTTPPeer) SetSendAddr(sendAddr string) {
	t.sendAddr = sendAddr
}

func (t *HTTPPeer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	msg, err := ReadMessage(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	t.in <- msg
	w.WriteHeader(http.StatusOK)
}

func (t *HTTPPeer) Send(ctx context.Context, msg Message) error {
	for range retries {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.sendAddr+"/recv", bytes.NewReader(msg.Bytes()))
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

func (t *HTTPPeer) Recv(ctx context.Context) (Message, error) {
	select {
	case msg := <-t.in:
		return msg, nil
	case <-ctx.Done():
		return Message{}, ctx.Err()
	}
}
