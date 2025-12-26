package factorfight

import (
	"context"
	"ztg/dice"
)

type ChannelPeer struct {
	in  chan Turn
	out chan Turn

	dicePeer dice.Peer
}

var _ Peer = ChannelPeer{}

func NewChannelPeers(dicePeer1, dicePeer2 dice.Peer) (ChannelPeer, ChannelPeer) {
	in1 := make(chan Turn, 1)
	in2 := make(chan Turn, 1)

	return ChannelPeer{in: in1, out: in2, dicePeer: dicePeer2}, ChannelPeer{in: in2, out: in1, dicePeer: dicePeer1}
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

func (c ChannelPeer) Dice() dice.Peer {
	return c.dicePeer
}
