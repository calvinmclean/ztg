package factorfight

import (
	"context"
	"ztg/dice"
)

type ChannelPeer struct {
	in  chan Move
	out chan Move

	dicePeer dice.Peer
}

var _ Peer = ChannelPeer{}

func NewChannelPeers(dicePeer1, dicePeer2 dice.Peer) (ChannelPeer, ChannelPeer) {
	in1 := make(chan Move, 1)
	in2 := make(chan Move, 1)

	return ChannelPeer{in: in1, out: in2, dicePeer: dicePeer2}, ChannelPeer{in: in2, out: in1, dicePeer: dicePeer1}
}

// RecvMove implements Peer.
func (c ChannelPeer) RecvMove(context.Context) (Move, error) {
	return <-c.in, nil
}

// SendMove implements Peer.
func (c ChannelPeer) SendMove(ctx context.Context, move Move) error {
	c.out <- move
	return nil
}

func (c ChannelPeer) Dice() dice.Peer {
	return c.dicePeer
}
