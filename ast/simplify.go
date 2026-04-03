package ast

import (
	"xex/object"

	ir_operator "xex/ast/operator"
)

func simplifyExpressionSlice(expressions []ExpressionNode) []ExpressionNode {
	if expressions == nil {
		return nil
	}
	out := make([]ExpressionNode, len(expressions))
	for i, expression := range expressions {
		out[i] = expression.Simplify()
	}
	return out
}

func simplifyExpressionPairs(pairs [][2]ExpressionNode) [][2]ExpressionNode {
	if pairs == nil {
		return nil
	}
	out := make([][2]ExpressionNode, len(pairs))
	for i, pair := range pairs {
		out[i] = [2]ExpressionNode{pair[0].Simplify(), pair[1].Simplify()}
	}
	return out
}

func foldPrefixLiteral(op ir_operator.Operator, value any) (_ ExpressionNode, ok bool) {
	boxed := object.BoxLit(value)
	result := object.DispatchPrefixOp(op)(boxed)
	return Literial{Value: object.UnBoxLit(result)}, true
}

func foldInfixLiteral(op ir_operator.Operator, left any, right any) (_ ExpressionNode, ok bool) {
	leftBox := object.BoxLit(left)
	rightBox := object.BoxLit(right)
	result := object.DispatchInfixOp(op)(leftBox, rightBox)
	return Literial{Value: object.UnBoxLit(result)}, true
}
