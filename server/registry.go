package server

import (
	"context"
	"sync"

	gamepb "ztg/gen/go/game/v1"

	"google.golang.org/grpc"
)

type challengeFunc func(ctx context.Context, conn *grpc.ClientConn) (*gamepb.ChallengeResponse, error)

type GameService interface {
	Name() string
	Register(server *grpc.Server)
	Challenge(ctx context.Context, conn *grpc.ClientConn) (*gamepb.ChallengeResponse, error)
}

type registry struct {
	mu    sync.RWMutex
	games map[string]GameService
}

func newRegistry() *registry {
	return &registry{
		games: make(map[string]GameService),
	}
}

func (r *registry) registerGame(service GameService) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.games[service.Name()] = service
}

func (r *registry) challenge(name string) (challengeFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	g, ok := r.games[name]
	if !ok {
		return nil, false
	}

	return g.Challenge, true
}
