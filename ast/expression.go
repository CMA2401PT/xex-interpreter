package ast

import (
	"fmt"
	"strings"
	ir_operator "xex/ast/operator"
)

type Literial struct {
	Value any
}

func (l Literial) ExpressionNode() {}
func (l Literial) Simplify() ExpressionNode {
	return l
}
func (l Literial) String() string { return l.IdentString(0) }
func (l Literial) IdentString(identLevel int) string {
	return fmt.Sprintf("%v", l.Value)
}

type ListLiteral struct {
	Elements []ExpressionNode
}

func (ll ListLiteral) ExpressionNode() {}
func (ll ListLiteral) Simplify() ExpressionNode {
	ll.Elements = simplifyExpressionSlice(ll.Elements)
	return ll
}
func (ll ListLiteral) String() string { return ll.IdentString(0) }
func (ll ListLiteral) IdentString(identLevel int) string {
	elements := []string{}
	for _, el := range ll.Elements {
		elements = append(elements, el.IdentString(identLevel))
	}
	return "[" + strings.Join(elements, ",") + "]"
}

type MapLiteral struct {
	Pairs [][2]ExpressionNode
}

func (ml MapLiteral) ExpressionNode() {}
func (ml MapLiteral) Simplify() ExpressionNode {
	ml.Pairs = simplifyExpressionPairs(ml.Pairs)
	return ml
}
func (ml MapLiteral) String() string { return ml.IdentString(0) }
func (ml MapLiteral) IdentString(identLevel int) string {
	pairs := []string{}
	for _, kv := range ml.Pairs {
		pairs = append(pairs, kv[0].IdentString(identLevel)+":"+kv[1].IdentString(identLevel))
	}
	return "{" + strings.Join(pairs, ",") + "}"
}

type YieldExpression struct {
	ReasonValue ExpressionNode
}

func (rs YieldExpression) ExpressionNode() {}
func (rs YieldExpression) Simplify() ExpressionNode {
	rs.ReasonValue = rs.ReasonValue.Simplify()
	return rs
}
func (rs YieldExpression) String() string { return rs.IdentString(0) }
func (rs YieldExpression) IdentString(identLevel int) string {
	return "yield(" + rs.ReasonValue.IdentString(identLevel) + ")"
}

type PrefixExpression struct {
	Operator ir_operator.Operator
	Right    ExpressionNode
}

func (pe PrefixExpression) ExpressionNode() {}
func (pe PrefixExpression) Simplify() ExpressionNode {
	pe.Right = pe.Right.Simplify()
	if right, ok := pe.Right.(Literial); ok {
		if simplified, ok := foldPrefixLiteral(pe.Operator, right.Value); ok {
			return simplified
		}
	}
	return pe
}
func (pe PrefixExpression) String() string { return pe.IdentString(0) }
func (pe PrefixExpression) IdentString(identLevel int) string {
	return "(" + pe.Operator.String() + pe.Right.IdentString(identLevel) + ")"
}

type InfixExpression struct {
	Operator    ir_operator.Operator
	Left, Right ExpressionNode
}

func (ie InfixExpression) ExpressionNode() {}
func (ie InfixExpression) Simplify() ExpressionNode {
	ie.Left = ie.Left.Simplify()
	ie.Right = ie.Right.Simplify()
	left, leftOk := ie.Left.(Literial)
	right, rightOk := ie.Right.(Literial)
	if leftOk && rightOk {
		if simplified, ok := foldInfixLiteral(ie.Operator, left.Value, right.Value); ok {
			return simplified
		}
	}
	return ie
}
func (ie InfixExpression) String() string { return ie.IdentString(0) }
func (ie InfixExpression) IdentString(identLevel int) string {
	return "(" + ie.Left.IdentString(identLevel) + ie.Operator.String() + ie.Right.IdentString(identLevel) + ")"
}
