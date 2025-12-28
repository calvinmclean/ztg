package factorfight

import "testing"

// func TestMoves(t *testing.T) {
// 	moves := generateMoves(72, 84, 2, 3)
// 	fmt.Println(len(moves))

// 	for _, m := range moves {
// 		fmt.Printf(
// 			"P1: %s = %d | P2: %s = %d\n",
// 			m.Pawn1.Expr, m.Pawn1.Result,
// 			m.Pawn2.Expr, m.Pawn2.Result,
// 		)
// 	}
// }

func TestExpr(t *testing.T) {
	expr := Expr{
		Op:    Mul,
		Right: V(3),
		Left: &Expr{
			Op:    Div,
			Left:  V(84),
			Right: V(2),
		},
	}

	res, err := expr.Eval()
	if err != nil {
		t.Fatal(err)
	}
	if res != 126 {
		t.Fatalf("got %d", res)
	}
}
