package highroll

import (
	"context"

	"ztg/dice"
	dicepb "ztg/gen/go/dice/v1"
	gamepb "ztg/gen/go/game/v1"
	"ztg/identity"
	"ztg/server"

	"google.golang.org/grpc"
)

const GameID = "ztg.HighRoll.v1"

type Service struct {
	dicepb.UnimplementedDiceServiceServer
	keyManager *identity.KeyManager
	serverAddr string
}

func NewService(keyManager *identity.KeyManager, serverAddr string) *Service {
	return &Service{
		keyManager: keyManager,
		serverAddr: serverAddr,
	}
}

func (s *Service) ID() string {
	return GameID
}

func (s *Service) Register(server *grpc.Server) {
	dicepb.RegisterDiceServiceServer(server, s)
}

// Challenge implements head-to-head dice for GameService registry
func (s *Service) Challenge(ctx context.Context, conn *grpc.ClientConn) (*gamepb.ChallengeResponse, error) {
	client := dicepb.NewDiceServiceClient(conn)
	stream, err := client.Roll(ctx)
	if err != nil {
		return nil, err
	}
	signer, verifier := server.CreateSignerVerifierPair(s.keyManager, s.serverAddr)
	highrollPeer := newPeer(stream, signer, verifier)
	roller, err := newRoller(highrollPeer, server.DefaultDieSides)
	if err != nil {
		return nil, err
	}
	roll, err := roller.RollSync(stream.Context(), 2)
	if err != nil {
		return nil, err
	}
	win := roll[0] > roll[1]
	msg := ""
	if roll[0] == roll[1] {
		msg = "Draw!"
	}
	_ = stream.CloseSend()
	return &gamepb.ChallengeResponse{Win: &win, Message: msg}, nil
}

// gRPC implementation for rolling dice
func (s *Service) Roll(stream dicepb.DiceService_RollServer) error {
	signer, verifier := server.CreateSignerVerifierPair(s.keyManager, s.serverAddr)
	highrollPeer := newPeer(stream, signer, verifier)
	roller, err := newRoller(highrollPeer, server.DefaultDieSides)
	if err != nil {
		return err
	}
	_, err = roller.RollSync(stream.Context(), 1)
	return err
}

func newRoller(peer dice.Peer, sides int) (dice.Roller, error) {
	if sides <= 0 {
		sides = server.DefaultDieSides
	}
	return dice.NewRoller(uint8(sides), peer)
}
