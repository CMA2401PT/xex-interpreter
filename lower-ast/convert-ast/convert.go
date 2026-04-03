package convert_ast

import (
	"fmt"
	"xex/ast"
	lower_ast "xex/lower-ast"
	convert_expression "xex/lower-ast/convert-expression"
	convert_statement "xex/lower-ast/convert-statement"
)

func ConvertFn(fn ast.CreateEnclosureExpression) lower_ast.FunctionProto {
	return lower_ast.NewFuncProto(
		fn.ParameterSymbols, fn.LocalSymbols, fn.FreeSymbols, fn.ImportSymbols, ConvertAst(fn.Body).(lower_ast.LowerAst),
		fn.NonAsync,
	)
}

func ConvertAst(a any) any {
	switch astt := a.(type) {
	default:
		panic(fmt.Sprintf("unknown ast type %T%v", astt, astt))
	case ast.CreateEnclosureExpression:
		proto := ConvertFn(astt)
		return lower_ast.CreateEnclosureExpression{FunctionProto: proto}
	case ast.StatementNode:
		graph := convert_statement.StatementToCFGraph(astt)
		cvtGraph := make(lower_ast.LowerAst, len(graph))
		for blockI, origBlock := range graph {
			cvtSeq := convert_expression.ConvertExpressionsKeepLast(origBlock.Sequence, ConvertFn)
			cvtGraph[blockI] = lower_ast.CFGBlock[lower_ast.SuffixOperatesSequence]{
				Sequence:     cvtSeq,
				IsNoCondJump: origBlock.IsNoCondJump,
				JumpTo:       origBlock.JumpTo,
				Aux:          origBlock.Aux,
			}
		}
		return cvtGraph
	case ast.ExpressionNode:
		astt = astt.Simplify()
		return convert_expression.ConvertExpression(astt, ConvertFn)
	}
}
