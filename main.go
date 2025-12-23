package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"ztg/dice"
)

func main() {
	sides := uint8(10)

	addr := os.Getenv("ADDR")
	peerAddr := os.Getenv("PEER_ADDR")

	if addr != "" && peerAddr != "" {
		runHTTP(sides, addr, peerAddr)
	} else {
		runSimple(sides)
	}
}

func runHTTP(sides uint8, addr, peerAddr string) {
	tp := dice.NewHTTPPeer(peerAddr)
	d := dice.NewRoller(addr, sides, tp)

	s := http.Server{Addr: addr, Handler: tp}
	go s.ListenAndServe()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	roll := d.Roll(ctx)

	out, err := roll.Result()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(out)
	s.Close()
}

// runSimple runs a single roll using two in-memory Rollers with ChannelPeers
func runSimple(sides uint8) {
	peer1, peer2 := dice.NewChannelPeers()

	d1 := dice.NewRoller("One", sides, peer1)
	d2 := dice.NewRoller("Two", sides, peer2)

	ctx := context.Background()
	roll1 := d1.Roll(ctx)
	roll2 := d2.Roll(ctx)

	r1, _ := roll1.Result()
	r2, _ := roll2.Result()

	if r1 != r2 {
		panic("invalid roll")
	}

	fmt.Println(r1)
}
