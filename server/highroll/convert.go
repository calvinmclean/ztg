package highroll

import (
	"github.com/calvinmclean/ztg/dice"
	dicepb "github.com/calvinmclean/ztg/gen/go/dice/v1"
)

func convertdicepbMessageToInternal(msg *dicepb.Message) dice.Message {
	return dice.Message{
		Type: dice.MsgType(msg.MsgType),
		Data: func(data []byte) [32]byte {
			var fixed [32]byte
			copy(fixed[:], data[:])
			return fixed
		}(msg.Data),
	}
}

func convertInternalDiceMessageToProto(msg dice.Message) *dicepb.Message {
	return &dicepb.Message{
		MsgType: dicepb.MessageType(msg.Type),
		Data:    msg.Data[:],
	}
}
