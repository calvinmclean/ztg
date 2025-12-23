package dice

import "context"

type ChannelPeer struct {
	in  chan Message
	out chan Message
}

// NewChannelPeers creates two Peers that use channels to communicate. This creates two
// instead of one because it only works if the Peers share channels in the correct way and
// also only works when running the Peers in the same program
func NewChannelPeers() (ChannelPeer, ChannelPeer) {
	in1 := make(chan Message, 1)
	in2 := make(chan Message, 1)

	return ChannelPeer{in: in1, out: in2}, ChannelPeer{in: in2, out: in1}
}

var _ Peer = ChannelPeer{}

// Recv implements Peer.
func (c ChannelPeer) Recv(context.Context) (Message, error) {
	return <-c.in, nil
}

// Send implements Peer.
func (c ChannelPeer) Send(ctx context.Context, msg Message) error {
	c.out <- msg
	return nil
}
