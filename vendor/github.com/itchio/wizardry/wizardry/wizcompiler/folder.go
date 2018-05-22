package wizcompiler

import (
	"fmt"
)

// This package implements constant folding

type Operator int

const (
	OperatorMul Operator = iota
	OperatorDiv
	OperatorBinaryAnd
	OperatorAdd
	OperatorSub
)

func (op Operator) Precedence() int {
	switch op {
	case OperatorMul, OperatorDiv, OperatorBinaryAnd:
		return 5
	case OperatorAdd, OperatorSub:
		return 4
	default:
		return 0
	}
}

func (op Operator) IsAssociative() bool {
	switch op {
	case OperatorMul, OperatorAdd:
		return true
	default:
		return false
	}
}

func (op Operator) Evaluate(lhs int64, rhs int64) int64 {
	switch op {
	case OperatorMul:
		return lhs * rhs
	case OperatorDiv:
		return lhs / rhs
	case OperatorBinaryAnd:
		return lhs & rhs
	case OperatorAdd:
		return lhs + rhs
	case OperatorSub:
		return lhs - rhs
	default:
		return -1
	}
}

func (op Operator) String() string {
	switch op {
	case OperatorMul:
		return "*"
	case OperatorDiv:
		return "/"
	case OperatorBinaryAnd:
		return "&"
	case OperatorAdd:
		return "+"
	case OperatorSub:
		return "-"
	default:
		return "?"
	}
}

type Expression interface {
	String() string
	Fold() Expression
}

type NumberLiteral struct {
	Value int64
}

var _ Expression = (*NumberLiteral)(nil)

func (nl *NumberLiteral) String() string {
	return fmt.Sprintf("%d", nl.Value)
}

func (nl *NumberLiteral) Fold() Expression {
	return nl
}

type VariableAccess struct {
	Name string
}

var _ Expression = (*VariableAccess)(nil)

func (va *VariableAccess) String() string {
	return va.Name
}

func (va *VariableAccess) Fold() Expression {
	return va
}

type BinaryOp struct {
	Operator Operator
	LHS      Expression
	RHS      Expression
}

var _ Expression = (*BinaryOp)(nil)

func (bo *BinaryOp) String() string {
	if rhs, ok := bo.RHS.(*BinaryOp); ok && rhs.Operator.Precedence() < bo.Operator.Precedence() {
		return fmt.Sprintf("%s%s(%s)", bo.LHS, bo.Operator, bo.RHS)
	}
	if lhs, ok := bo.LHS.(*BinaryOp); ok && lhs.Operator.Precedence() < bo.Operator.Precedence() {
		return fmt.Sprintf("(%s)%s%s", bo.LHS, bo.Operator, bo.RHS)
	}
	return fmt.Sprintf("%s%s%s", bo.LHS, bo.Operator, bo.RHS)
}

func (bo *BinaryOp) Fold() Expression {
	lhs := bo.LHS.Fold()
	rhs := bo.RHS.Fold()

	if bo.Operator == OperatorAdd {
		if ln, ok := lhs.(*NumberLiteral); ok && ln.Value == 0 {
			return rhs
		}
		if rn, ok := rhs.(*NumberLiteral); ok && rn.Value == 0 {
			return lhs
		}
	}

	if bo.Operator == OperatorSub {
		if ln, ok := lhs.(*NumberLiteral); ok && ln.Value == 0 {
			if rn, ok := rhs.(*NumberLiteral); ok {
				return &NumberLiteral{-rn.Value}
			}
		}
		if rn, ok := rhs.(*NumberLiteral); ok && rn.Value == 0 {
			return rhs
		}
	}

	if bo.Operator == OperatorMul {
		if ln, ok := lhs.(*NumberLiteral); ok && ln.Value == 0 {
			return &NumberLiteral{0}
		}
		if rn, ok := rhs.(*NumberLiteral); ok && rn.Value == 0 {
			return &NumberLiteral{0}
		}
	}

	if ln, ok := lhs.(*NumberLiteral); ok {
		if rn, ok := rhs.(*NumberLiteral); ok {
			return &NumberLiteral{
				Value: bo.Operator.Evaluate(ln.Value, rn.Value),
			}
		}

		if rop, ok := rhs.(*BinaryOp); ok && rop.Operator == bo.Operator && bo.Operator.IsAssociative() {
			if cln, ok := rop.LHS.(*NumberLiteral); ok {
				return &BinaryOp{
					LHS:      &NumberLiteral{bo.Operator.Evaluate(ln.Value, cln.Value)},
					RHS:      rop.RHS.Fold(),
					Operator: bo.Operator,
				}
			} else if crn, ok := rop.RHS.(*NumberLiteral); ok {
				return &BinaryOp{
					LHS:      rop.LHS.Fold(),
					RHS:      &NumberLiteral{bo.Operator.Evaluate(ln.Value, crn.Value)},
					Operator: bo.Operator,
				}
			}
		}
	} else if rn, ok := rhs.(*NumberLiteral); ok {
		if lop, ok := lhs.(*BinaryOp); ok && lop.Operator == bo.Operator && bo.Operator.IsAssociative() {
			if cln, ok := lop.LHS.(*NumberLiteral); ok {
				return &BinaryOp{
					LHS:      &NumberLiteral{bo.Operator.Evaluate(rn.Value, cln.Value)},
					RHS:      lop.RHS.Fold(),
					Operator: bo.Operator,
				}
			} else if crn, ok := lop.RHS.(*NumberLiteral); ok {
				return &BinaryOp{
					LHS:      lop.LHS.Fold(),
					RHS:      &NumberLiteral{bo.Operator.Evaluate(rn.Value, crn.Value)},
					Operator: bo.Operator,
				}
			}
		}
	}

	return &BinaryOp{
		LHS:      lhs,
		Operator: bo.Operator,
		RHS:      rhs,
	}
}
