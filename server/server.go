package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"ztg/dice"
	"ztg/factorfight"
	"ztg/identity"
	protodice "ztg/proto/dice"
	protofactorfight "ztg/proto/factorfight"
	gamepb "ztg/proto/game"

	"github.com/rs/xid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// TODO: Add Identity for self server

// GRPCServer represents a gRPC server instance.
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
}

// GRPCServerConfig holds configuration for initializing a gRPC server.
type GRPCServerConfig struct {
	Addr string
}

// gameService implements the GameService RPC defined in game.proto.
type gameService struct {
	gamepb.UnimplementedGameServiceServer

	diceMap        *sync.Map
	factorfightMap *sync.Map
}

func (s *gameService) Challenge(ctx context.Context, req *gamepb.ChallengeRequest) (*gamepb.ChallengeResponse, error) {
	conn, err := grpc.NewClient(
		req.Target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	initClient := gamepb.NewGameServiceClient(conn)

	switch strings.ToLower(req.GameName) {
	case "factorfight":
		resp, err := initClient.InitiateGame(ctx, &gamepb.InitiateGameRequest{
			PlayerIdentity: &gamepb.Identity{
				Name:    "Challenger",
				Address: "localhost:50052", // TODO
			},
			GameName: req.GameName,
		})
		if err != nil {
			return nil, err
		}

		if !resp.Accepted {
			return nil, errors.New("game declined")
		}

		diceClient := protodice.NewRollerServiceClient(conn)
		ffClient := protofactorfight.NewFactorFightServiceClient(conn)

		diceIn := make(chan dice.Message, 1)
		factorfightIn := make(chan factorfight.Move, 1)
		s.diceMap.Store(resp.SessionId, diceIn)
		s.factorfightMap.Store(resp.SessionId, factorfightIn)

		dicePeer, err := NewDicePeer(diceClient, diceIn, resp.SessionId)
		if err != nil {
			return nil, err
		}

		peer, err := NewFactorfightPeer(identity.Identity{
			Name:    "Opponent",
			Address: req.Target,
		}, dicePeer, ffClient, factorfightIn, resp.SessionId)
		if err != nil {
			return nil, err
		}

		session, err := factorfight.NewSession(resp.SessionId, identity.Identity{
			Name:    "Challenger",
			Address: "localhost:50052", // TODO
		}, peer, factorfight.DefaultStrategy)
		if err != nil {
			return nil, err
		}

		go func() {
			defer conn.Close()
			_, log, err := session.PlayWithInitiative(context.Background(), false)
			if err != nil {
				panic(err)
			}

			fmt.Println(log)
		}()
	}

	return &gamepb.ChallengeResponse{}, nil
}

// InitiateGame handles requests to initiate a game.
func (s *gameService) InitiateGame(ctx context.Context, req *gamepb.InitiateGameRequest) (*gamepb.InitiateGameResponse, error) {
	conn, err := grpc.NewClient(
		req.PlayerIdentity.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	// TODO: put at struct level
	self := identity.Identity{Name: "Self"}

	switch strings.ToLower(req.GameName) {
	case "factorfight":
		diceClient := protodice.NewRollerServiceClient(conn)
		ffClient := protofactorfight.NewFactorFightServiceClient(conn)

		sessionID := xid.New().String()

		diceIn := make(chan dice.Message, 1)
		factorfightIn := make(chan factorfight.Move, 1)
		s.diceMap.Store(sessionID, diceIn)
		s.factorfightMap.Store(sessionID, factorfightIn)

		dicePeer, err := NewDicePeer(diceClient, diceIn, sessionID)
		if err != nil {
			return nil, err
		}

		peer, err := NewFactorfightPeer(identity.Identity{
			Name:    req.PlayerIdentity.Name,
			Address: req.PlayerIdentity.Address,
		}, dicePeer, ffClient, factorfightIn, sessionID)
		if err != nil {
			return nil, err
		}

		session, err := factorfight.NewSession(sessionID, self, peer, factorfight.DefaultStrategy)
		if err != nil {
			return nil, err
		}

		// TODO: do something better
		go func() {
			defer conn.Close()
			_, log, err := session.PlayWithInitiative(context.Background(), true)
			if err != nil {
				panic(err)
			}

			fmt.Println(log)
		}()

		return &gamepb.InitiateGameResponse{
			SessionId: session.ID,
			Accepted:  true,
			Message:   "",
		}, nil
	}

	return &gamepb.InitiateGameResponse{
		Accepted: true,
		Message:  "Game request accepted!",
	}, nil
}

// NewGRPCServer initializes a new GRPC server.
func NewGRPCServer(cfg GRPCServerConfig) (*GRPCServer, error) {
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind gRPC server on %s: %w", cfg.Addr, err)
	}

	diceMap := &sync.Map{}
	ffMap := &sync.Map{}

	diceServer := NewDiceServer(diceMap)
	ffserver := NewFactorfightServer(ffMap)

	server := grpc.NewServer()
	protodice.RegisterRollerServiceServer(server, diceServer)
	protofactorfight.RegisterFactorFightServiceServer(server, ffserver)
	gamepb.RegisterGameServiceServer(server, newGameService(diceMap, ffMap))

	reflection.Register(server)

	ctx, cancel := context.WithCancel(context.Background())

	return &GRPCServer{
		server:   server,
		listener: listener,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func newGameService(diceMap, ffMap *sync.Map) *gameService {
	return &gameService{
		diceMap:        diceMap,
		factorfightMap: ffMap,
	}
}

// Run starts the gRPC server.
func (g *GRPCServer) Run() error {
	return g.server.Serve(g.listener)
}

// Stop gracefully stops the gRPC server.
func (g *GRPCServer) Stop() {
	g.cancel()
	g.server.GracefulStop()
}
