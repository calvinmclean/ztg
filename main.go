package main

import (
	"fmt"

	"ztg/dice"
)

func main() {
	d1 := dice.NewRoller("One")
	d2 := dice.NewRoller("Two")

	roll1 := d1.Roll(d2)
	roll2 := d2.Roll(d1)

	r1, _ := roll1.Result()
	r2, _ := roll2.Result()

	if r1 != r2 {
		panic("invalid roll")
	}

	fmt.Println(r1)
}
