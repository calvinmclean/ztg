package factorfight

import (
	"fmt"
	"strconv"
)

type Move struct {
	Pawn1 PawnMove
	Pawn2 PawnMove
}

type PawnMove struct {
	Expr   *Expr
	Result int
}

type Op string

const (
	Add Op = "+"
	Sub Op = "-"
	Mul Op = "*"
	Div Op = "/"
)

type Expr struct {
	Op    Op
	Left  *Expr
	Right *Expr
	Value *int
}

// EvalWithMax will evaluate the expression and return true if it is valid and < max
func (e *Expr) EvalWithMax(max int) (int, bool) {
	r, err := e.Eval()
	if err != nil {
		return 0, false
	}

	if r > max {
		return 0, false
	}

	return r, true
}

func (e *Expr) Eval() (int, error) {
	if e.Value != nil {
		return *e.Value, nil
	}

	l, err := e.Left.Eval()
	if err != nil {
		return 0, err
	}

	r, err := e.Right.Eval()
	if err != nil {
		return 0, err
	}

	switch e.Op {
	case Add:
		return l + r, nil
	case Sub:
		return l - r, nil
	case Mul:
		return l * r, nil
	case Div:
		if r == 0 || l%r != 0 {
			return 0, fmt.Errorf("invalid division")
		}
		return l / r, nil
	default:
		return 0, fmt.Errorf("unknown op")
	}
}

func (e *Expr) String() string {
	if e.Value != nil {
		return strconv.Itoa(*e.Value)
	}
	return fmt.Sprintf("(%s %s %s)", e.Left, e.Op, e.Right)
}

func V(n int) *Expr {
	return &Expr{Value: &n}
}

func pawnMovesZeroDice(start int) []PawnMove {
	expr := V(start)
	return []PawnMove{
		{Expr: expr, Result: start},
	}
}

func pawnMovesOneDie(start, die int) []PawnMove {
	ops := []Op{Add, Sub, Mul, Div}
	var moves []PawnMove

	for _, op := range ops {
		expr := &Expr{
			Op:    op,
			Left:  V(start),
			Right: V(die),
		}

		if res, ok := expr.EvalWithMax(goal); ok {
			moves = append(moves, PawnMove{Expr: expr, Result: res})
		}
	}

	return moves
}

func pawnMovesTwoDice(start, d1, d2 int) []PawnMove {
	ops := []Op{Add, Sub, Mul, Div}
	dice := [][2]int{{d1, d2}, {d2, d1}}

	var moves []PawnMove

	for _, pair := range dice {
		a, b := pair[0], pair[1]

		for _, op1 := range ops {
			for _, op2 := range ops {

				// (start op1 a) op2 b
				expr1 := &Expr{
					Op: op2,
					Left: &Expr{
						Op:    op1,
						Left:  V(start),
						Right: V(a),
					},
					Right: V(b),
				}

				if res, ok := expr1.EvalWithMax(goal); ok {
					moves = append(moves, PawnMove{Expr: expr1, Result: res})
				}

				// Order of operations does not matter for all add/subtract
				if (op1 == Add || op1 == Sub) && (op2 == Add || op2 == Sub) {
					continue
				}

				// Order of operations does not matter for all Multiply
				if op1 == Mul && op2 == Mul {
					continue
				}

				// start op1 (a op2 b)
				expr2 := &Expr{
					Op:   op1,
					Left: V(start),
					Right: &Expr{
						Op:    op2,
						Left:  V(a),
						Right: V(b),
					},
				}

				if res, ok := expr2.EvalWithMax(goal); ok {
					moves = append(moves, PawnMove{Expr: expr2, Result: res})
				}
			}
		}
	}

	return moves
}

type diceAllocation struct {
	Pawn1 []int
	Pawn2 []int
}

func allAllocations(d1, d2 int) []diceAllocation {
	return []diceAllocation{
		// {P1: []int{}, P2: []int{d1, d2}},
		{Pawn1: []int{d1, d2}, Pawn2: []int{}},
		// {P1: []int{d1}, P2: []int{d2}},
		// {P1: []int{d2}, P2: []int{d1}},
	}
}

func pawnMoves(start int, dice []int) []PawnMove {
	switch len(dice) {
	case 0:
		return pawnMovesZeroDice(start)
	case 1:
		return pawnMovesOneDie(start, dice[0])
	case 2:
		return pawnMovesTwoDice(start, dice[0], dice[1])
	default:
		return nil
	}
}

func generateMoves(p1, p2 int, d1, d2 int) []Move {
	var moves []Move

	for _, alloc := range allAllocations(d1, d2) {
		p1Moves := pawnMoves(p1, alloc.Pawn1)
		p2Moves := pawnMoves(p2, alloc.Pawn2)

		for _, m1 := range p1Moves {
			for _, m2 := range p2Moves {
				// no bumping (yet)
				if m1.Result == m2.Result {
					continue
				}

				// one pawn must move
				if m1.Result == p1 && m2.Result == p2 {
					continue
				}

				// must not go past goal
				if m1.Result > goal || m2.Result > goal {
					continue
				}

				moves = append(moves, Move{
					Pawn1: m1,
					Pawn2: m2,
				})
			}
		}
	}

	return moves
}
