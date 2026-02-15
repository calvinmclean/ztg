package highroll

import (
	"context"

	"github.com/calvinmclean/ztg/dice"
	dicepb "github.com/calvinmclean/ztg/gen/go/proto/dice/v1"
	gamepb "github.com/calvinmclean/ztg/gen/go/proto/game/v1"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/identity/store"
	"github.com/calvinmclean/ztg/server"

	"google.golang.org/grpc"
)

const GameID = "ztg.HighRoll.v1"

type Service struct {
	dicepb.UnimplementedDiceServiceServer
	keyManager *identity.KeyManager
	serverAddr string
	store      store.Store // SQL store for persistent identity storage
}

func NewService(keyManager *identity.KeyManager, serverAddr string, sqlStore store.Store) *Service {
	return &Service{
		keyManager: keyManager,
		serverAddr: serverAddr,
		store:      sqlStore,
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
	signer, verifier := server.CreateSignerVerifierPair(s.keyManager, s.serverAddr, s.store)
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
	signer, verifier := server.CreateSignerVerifierPair(s.keyManager, s.serverAddr, s.store)
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
