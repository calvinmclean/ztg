package factorfight

import "context"

type ChannelPeer struct {
	in  chan Turn
	out chan Turn
}

var _ Peer = ChannelPeer{}

func NewChannelPeers() (ChannelPeer, ChannelPeer) {
	in1 := make(chan Turn, 1)
	in2 := make(chan Turn, 1)

	return ChannelPeer{in: in1, out: in2}, ChannelPeer{in: in2, out: in1}
}

// RecvTurn implements Peer.
func (c ChannelPeer) RecvTurn(context.Context) (Turn, error) {
	return <-c.in, nil
}

// SendTurn implements Peer.
func (c ChannelPeer) SendTurn(ctx context.Context, turn Turn) error {
	c.out <- turn
	return nil
}
