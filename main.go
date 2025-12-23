package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	d, _ := dice.NewRoller(addr, sides, tp)

	s := http.Server{Addr: addr, Handler: tp}
	go s.ListenAndServe()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Running... press Ctrl+C to stop")

	ctx, bigCancel := context.WithCancel(context.Background())
	defer bigCancel()

	defer s.Close()

	go func() {
		<-sigCh
		fmt.Println("\nCtrl+C received, exiting loop")
		bigCancel()
	}()

	for {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)

		roll := d.Roll(ctx)
		out, err := roll.Get(8)
		cancel()

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(out)

		time.Sleep(1 * time.Second)
	}
}

// runSimple runs a single roll using two in-memory Rollers with ChannelPeers
func runSimple(sides uint8) {
	peer1, peer2 := dice.NewChannelPeers()

	d1, _ := dice.NewRoller("One", sides, peer1)
	d2, _ := dice.NewRoller("Two", sides, peer2)

	ctx := context.Background()
	roll1 := d1.Roll(ctx)
	roll2 := d2.Roll(ctx)

	r1, _ := roll1.GetOne()
	r2, _ := roll2.GetOne()

	if r1 != r2 {
		panic("invalid roll")
	}

	fmt.Println(r1)
}
