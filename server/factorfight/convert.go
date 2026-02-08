package factorfight

import (
	"github.com/calvinmclean/ztg/dice"
	"github.com/calvinmclean/ztg/factorfight"
	dicepb "github.com/calvinmclean/ztg/gen/go/proto/dice/v1"
	factorfightpb "github.com/calvinmclean/ztg/gen/go/proto/factorfight/v1"
)

// convertInternalFactorfightMoveToProto converts factorfight.Move to factorfightpb.Move.
func convertInternalFactorfightMoveToProto(move factorfight.Move) *factorfightpb.Move {
	toProtoPawn := func(p factorfight.PawnMove) *factorfightpb.PawnMove {
		return &factorfightpb.PawnMove{
			Result: int32(p.Result),
			Bump:   int32(p.Bump),
			Expr:   p.Expr.String(),
		}
	}
	return &factorfightpb.Move{
		Pawn1: toProtoPawn(move.Pawn1),
		Pawn2: toProtoPawn(move.Pawn2),
	}
}

func convertfactorfightpbMoveToInternal(move *factorfightpb.Move) factorfight.Move {
	// Conversion with safer embedding of Expr.
	return factorfight.Move{
		Pawn1: factorfight.PawnMove{
			Result: int(move.GetPawn1().GetResult()),
			Bump:   factorfight.BumpStatus(move.GetPawn1().GetBump()),
			Expr:   &factorfight.Expr{},
		},
		Pawn2: factorfight.PawnMove{
			Result: int(move.GetPawn2().GetResult()),
			Bump:   factorfight.BumpStatus(move.GetPawn2().GetBump()),
			// TODO: proto encode and decode Expr
			Expr: &factorfight.Expr{},
		},
	}
}

// convertdicepbMessageToInternal converts dicepb.Message to dice.Message.
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

// convertInternalDiceMessageToProto converts dice.Message to dicepb.Message.
func convertInternalDiceMessageToProto(msg dice.Message) *dicepb.Message {
	return &dicepb.Message{
		MsgType: dicepb.MessageType(msg.Type),
		Data:    msg.Data[:],
	}
}
