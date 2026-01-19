package factorfight

import (
	"context"
	"fmt"
	"ztg/factorfight"
	"ztg/identity"
	"ztg/server"

	factorfightpb "ztg/gen/go/factorfight/v1"
	gamepb "ztg/gen/go/game/v1"
	identitypb "ztg/gen/go/identity/v1"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Config struct {
	// Strategy controls how your implementation will choose which move to make after rolling
	Strategy factorfight.Strategy
	// OnGameComplete runs after your server receives a challenge from another player. The most basic/common
	// use for it would be notifying yourself of win/lose
	OnGameComplete func(win bool, log factorfight.GameLog)
}

// Service implements the gRPC server for FactorFight.
type Service struct {
	factorfightpb.UnimplementedFactorFightServiceServer

	cfg        Config
	keyManager *identity.KeyManager
	serverAddr string
	signedMode bool
}

// NewService creates a new FactorFight service.
func NewService(cfg Config, keyManager *identity.KeyManager, serverAddr string, signedMode bool) *Service {
	return &Service{
		cfg:        cfg,
		keyManager: keyManager,
		serverAddr: serverAddr,
		signedMode: signedMode,
	}
}

func (s Service) Name() string {
	return "factorfight"
}

func (s *Service) Register(server *grpc.Server) {
	factorfightpb.RegisterFactorFightServiceServer(server, s)
}

// StreamGame handles the gRPC streaming communication.
func (s *Service) Play(stream factorfightpb.FactorFightService_PlayServer) error {
	signer, verifier := server.CreateSignerVerifierPair(s.keyManager, s.serverAddr, s.signedMode)

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
		s.cfg.OnGameComplete(win, log)
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

	signer, verifier := server.CreateSignerVerifierPair(s.keyManager, s.serverAddr, s.signedMode)

	// Cache the peer's public key for signature verification
	if verifier != nil {
		verifier.AddPeerIdentity(conn.Target(), peerIdentity.PublicKey)
	}

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
		s.cfg.OnGameComplete(win, log)
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
