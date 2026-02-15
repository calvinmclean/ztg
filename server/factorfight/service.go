package factorfight

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/calvinmclean/ztg/factorfight"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/identity/store"
	"github.com/calvinmclean/ztg/server"

	factorfightpb "github.com/calvinmclean/ztg/gen/go/proto/factorfight/v1"
	gamepb "github.com/calvinmclean/ztg/gen/go/proto/game/v1"
	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const GameID = "ztg.FactorFight.v1"

type Config struct {
	// Strategy controls how your implementation will choose which move to make after rolling
	Strategy factorfight.Strategy
	// OnGameComplete runs after your server receives a challenge from another player. The most basic/common
	// use for it would be notifying yourself of win/lose
	OnGameComplete func(win bool, log factorfight.GameLog, logger *slog.Logger)
}

// Service implements the gRPC server for FactorFight.
type Service struct {
	factorfightpb.UnimplementedFactorFightServiceServer

	cfg        Config
	keyManager *identity.KeyManager
	serverAddr string
	store      store.Store // SQL store for persistent identity storage
	logger     *slog.Logger
}

// NewService creates a new FactorFight service.
func NewService(cfg Config, keyManager *identity.KeyManager, serverAddr string, sqlStore store.Store, logger *slog.Logger) *Service {
	return &Service{
		cfg:        cfg,
		keyManager: keyManager,
		serverAddr: serverAddr,
		store:      sqlStore,
		logger:     logger.With("service", GameID),
	}
}

func (s Service) ID() string {
	return GameID
}

func (s *Service) Register(server *grpc.Server) {
	factorfightpb.RegisterFactorFightServiceServer(server, s)
}

// StreamGame handles the gRPC streaming communication.
func (s *Service) Play(stream factorfightpb.FactorFightService_PlayServer) (err error) {
	defer func() {
		s.logger.Debug("completed request to Play", "err", err)
	}()
	s.logger.Debug("received request to Play")
	signer, verifier := server.CreateSignerVerifierPair(s.keyManager, s.serverAddr, s.store)

	factorfightPeer := createFactorfightPeer(stream, signer, verifier)

	strategy := resolveStrategy(s.cfg.Strategy)

	session, err := factorfight.NewSession(factorfightPeer, strategy)
	if err != nil {
		return err
	}

	win, log, err := session.PlayWithInitiative(stream.Context(), true)
	if err != nil {
		return err
	}

	if s.cfg.OnGameComplete != nil {
		s.cfg.OnGameComplete(win, log, s.logger)
	}

	return nil
}

func (s *Service) Challenge(ctx context.Context, conn *grpc.ClientConn) (*gamepb.ChallengeResponse, error) {
	ffClient := factorfightpb.NewFactorFightServiceClient(conn)
	stream, err := ffClient.Play(ctx)
	if err != nil {
		return nil, err
	}

	// Get peer identity for signature verification
	identityClient := identitypb.NewIdentityServiceClient(conn)
	peerIdentity, err := identityClient.GetIdentity(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("failed to get peer identity: %w", err)
	}

	signer, verifier := server.CreateSignerVerifierPair(s.keyManager, s.serverAddr, s.store)

	// Cache the peer's public key for signature verification
	verifier.AddPeerIdentity(conn.Target(), peerIdentity.PublicKey)

	factorfightPeer := createFactorfightPeer(stream, signer, verifier)

	strategy := resolveStrategy(s.cfg.Strategy)

	session, err := factorfight.NewSession(factorfightPeer, strategy)
	if err != nil {
		return nil, err
	}

	win, log, err := session.PlayWithInitiative(stream.Context(), false)
	if err != nil {
		return nil, err
	}

	if s.cfg.OnGameComplete != nil {
		s.cfg.OnGameComplete(win, log, s.logger)
	}

	err = stream.CloseSend()
	return &gamepb.ChallengeResponse{
		Win: &win,
	}, err
}

// resolveStrategy returns the provided strategy or the default strategy if nil
func resolveStrategy(strategy factorfight.Strategy) factorfight.Strategy {
	if strategy != nil {
		return strategy
	}
	return factorfight.DefaultStrategy
}
