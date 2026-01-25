package server

import (
	"context"
	"maps"
	"slices"
	"sync"

	gamepb "github.com/calvinmclean/ztg/gen/go/game/v1"

	"google.golang.org/grpc"
)

type challengeFunc func(ctx context.Context, conn *grpc.ClientConn) (*gamepb.ChallengeResponse, error)

type GameService interface {
	ID() string
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

	r.games[service.ID()] = service
}

func (r *registry) challenge(gameID string) (challengeFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	g, ok := r.games[gameID]
	if !ok {
		return nil, false
	}

	return g.Challenge, true
}

// GameIDs returns a list of registered game names.
func (r *registry) GameIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return slices.Collect(maps.Keys(r.games))
}
