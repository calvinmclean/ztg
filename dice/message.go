package dice

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MsgType uint8

const (
	MsgTypeUnkown MsgType = iota
	MsgTypeCommitment
	MsgTypeSecret
	MsgTypeRollConfirmation
)

func (m MsgType) String() string {
	switch m {
	case MsgTypeCommitment:
		return "Commitment"
	case MsgTypeRollConfirmation:
		return "RollConfirmation"
	case MsgTypeSecret:
		return "Secret"
	default:
		fallthrough
	case MsgTypeUnkown:
		return "Unknown"
	}
}

func (m MsgType) IsMsgType() bool {
	switch m {
	case MsgTypeCommitment, MsgTypeRollConfirmation, MsgTypeSecret:
		return true
	default:
		fallthrough
	case MsgTypeUnkown:
		return false
	}
}

// Message wraps the MsgType and data
type Message struct {
	Type MsgType
	Data [32]byte
}

// ReadMessage reads the Message from the io.Reader. It expects [33]byte with this layout:
// [MsgType(len=1)][Data(len=32)]
func ReadMessage(r io.Reader) (Message, error) {
	var result Message

	if err := binary.Read(r, binary.BigEndian, &result.Type); err != nil {
		return Message{}, err
	}
	if !result.Type.IsMsgType() {
		return Message{}, fmt.Errorf("unknown MessageType: %d", result.Type)
	}

	if _, err := io.ReadFull(r, result.Data[:]); err != nil {
		return Message{}, err
	}

	return result, nil
}

// Bytes converts the Message to a bytes format
func (m Message) Bytes() []byte {
	result := [33]byte{byte(m.Type)}
	copy(result[1:], m.Data[:])
	return result[:]
}
