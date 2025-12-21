package dice

import (
	"bytes"
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
		// TODO: is this correct? This makes it work before Recv is called
		in: make(chan [32]byte, 1),
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

func (t *HTTPTransport) Send(b [32]byte) {
	for range retries {
		resp, err := http.Post(
			t.sendAddr+"/recv",
			"application/octet-stream",
			bytes.NewReader(b[:]),
		)
		if err != nil {
			if errors.Is(err, syscall.ECONNREFUSED) {
				// connection was actively refused
				select {
				case <-time.After(retryDelay):
					// case <-ctx.Done():
					// 	return nil, ctx.Err()
				}
				continue
			}
			// otherwise return
			return
		}
		resp.Body.Close()

		break

	}
}

func (t *HTTPTransport) Recv() <-chan [32]byte {
	return t.in
}
