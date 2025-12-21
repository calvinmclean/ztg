package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

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
	tp := dice.NewHTTPTransport(peerAddr)
	d := dice.NewRoller(addr, sides, tp)

	s := http.Server{Addr: addr, Handler: tp}
	go s.ListenAndServe()

	roll := d.Roll()

	out, _ := roll.Result()

	fmt.Println(out)
	s.Close()
}

// runSimple runs a single roll using two in-memory Rollers with RWTransport
func runSimple(sides uint8) {
	rd1, wr1 := io.Pipe()
	rd2, wr2 := io.Pipe()
	tport1 := dice.NewRWTransport(rd1, wr2)
	tport2 := dice.NewRWTransport(rd2, wr1)

	d1 := dice.NewRoller("One", sides, tport1)
	d2 := dice.NewRoller("Two", sides, tport2)

	roll1 := d1.Roll()
	roll2 := d2.Roll()

	r1, _ := roll1.Result()
	r2, _ := roll2.Result()

	if r1 != r2 {
		panic("invalid roll")
	}

	fmt.Println(r1)
}
