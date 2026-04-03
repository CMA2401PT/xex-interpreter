package convert_expression

import (
	"fmt"
	ast "xex/ast"
	lower_ast "xex/lower-ast"
)

// 将 ast.ExpressionNode 转为 lower_ast.SuffixOperatesSequence
// 不允许使用 lower_ast.DropOperate, 都严格遵守 consume N produce 1 的规则
func ConvertExpression(
	expression ast.ExpressionNode,
	// 由于函数的转换本身会涉及 statement 的转换，所以我们这里将其摘出来
	fnConvert func(ast.CreateEnclosureExpression) lower_ast.FunctionProto,
) lower_ast.SuffixOperatesSequence {
	if expression == nil {
		panic("expression is nil")
	}
	expression = expression.Simplify()
	switch exp := expression.(type) {
	case ast.IdentifierExpression:
		return lower_ast.SuffixOperatesSequence{
			lower_ast.IdentifierExpression{
				IdentifierName: exp.Identifier.Value,
				Scope:          exp.Scope,
			},
		}
	case ast.Literial:
		return lower_ast.SuffixOperatesSequence{
			lower_ast.LiterialExpression{Value: exp.Value},
		}
	case ast.ListLiteral:
		out := ConvertExpressionsKeepAll(exp.Elements, fnConvert)
		out = append(out, lower_ast.ListExpression{ElementsCount: int32(len(exp.Elements))})
		return out
	case ast.MapLiteral:
		out := lower_ast.SuffixOperatesSequence{}
		for _, pair := range exp.Pairs {
			out = append(out, ConvertExpression(pair[0], fnConvert)...)
			out = append(out, ConvertExpression(pair[1], fnConvert)...)
		}
		out = append(out, lower_ast.MapExpression{PairsCount: int32(len(exp.Pairs))})
		return out
	case ast.YieldExpression:
		out := ConvertExpression(exp.ReasonValue, fnConvert)
		out = append(out, lower_ast.YieldExpression{})
		return out
	case ast.PrefixExpression:
		out := ConvertExpression(exp.Right, fnConvert)
		out = append(out, lower_ast.PrefixExpression{Operator: exp.Operator})
		return out
	case ast.InfixExpression:
		out := make(lower_ast.SuffixOperatesSequence, 0)
		out = append(out, ConvertExpression(exp.Left, fnConvert)...)
		out = append(out, ConvertExpression(exp.Right, fnConvert)...)
		out = append(out, lower_ast.InfixExpression{Operator: exp.Operator})
		return out
	case ast.AssignmentExpression:
		return convertAssignmentExpression(exp, fnConvert)
	case ast.CreateEnclosureExpression:
		proto := fnConvert(exp)
		return lower_ast.SuffixOperatesSequence{
			lower_ast.CreateEnclosureExpression{
				FunctionProto: proto,
			},
		}
	case ast.CallExpression:
		out := ConvertExpression(exp.Function, fnConvert)
		out = append(out, ConvertExpressionsKeepAll(exp.Arguments, fnConvert)...)
		out = append(out, lower_ast.CallExpression{ArgumentsCount: int32(len(exp.Arguments))})
		return out
	default:
		panic(fmt.Sprintf("unsupported expression node: %T", expression))
	}
}

// 将多个 ast.ExpressionNode 顺序 lower，并保留每个 expression 的结果值。
// 最终净产出 len(expressions) 个值。
func ConvertExpressionsKeepAll(
	expressions []ast.ExpressionNode,
	// 由于函数的转换本身会涉及 statement 的转换，所以我们这里将其摘出来
	fnConvert func(ast.CreateEnclosureExpression) lower_ast.FunctionProto,
) lower_ast.SuffixOperatesSequence {
	out := lower_ast.SuffixOperatesSequence{}
	for _, exp := range expressions {
		out = append(out, ConvertExpression(exp, fnConvert)...)
	}
	return out
}

// 将多个 ast.ExpressionNode 顺序 lower，并显式丢弃前面的结果，只保留最后一个值。
// 空切片会产生空序列。
func ConvertExpressionsKeepLast(
	expressions []ast.ExpressionNode,
	fnConvert func(ast.CreateEnclosureExpression) lower_ast.FunctionProto,
) lower_ast.SuffixOperatesSequence {
	out := lower_ast.SuffixOperatesSequence{}
	for i, exp := range expressions {
		if i > 0 {
			out = append(out, lower_ast.DropOperate{})
		}
		out = append(out, ConvertExpression(exp, fnConvert)...)
	}
	return out
}

func convertAssignmentExpression(
	expression ast.AssignmentExpression,
	fnConvert func(ast.CreateEnclosureExpression) lower_ast.FunctionProto,
) lower_ast.SuffixOperatesSequence {
	switch left := expression.Left.(type) {
	case ast.LeftIdentifier:
		out := ConvertExpression(expression.Right, fnConvert)
		out = append(out, lower_ast.AssignmentIdentifierExpression{
			Identifier: left.Identifier,
			Scope:      left.Scope,
		})
		return out
	case ast.LeftSetAttribute:
		out := ConvertExpression(left.CanSetAttribute, fnConvert)
		out = append(out, ConvertExpression(expression.Right, fnConvert)...)
		out = append(out, lower_ast.AssignmentAttributeExpression{Attribute: left.Attribute})
		return out
	case ast.LeftSetIndex:
		out := ConvertExpression(left.CanSetIndex, fnConvert)
		out = append(out, ConvertExpression(left.Index, fnConvert)...)
		out = append(out, ConvertExpression(expression.Right, fnConvert)...)
		out = append(out, lower_ast.AssignmentIndexExpression{})
		return out
	default:
		panic(fmt.Sprintf("unsupported assignment left value: %T", expression.Left))
	}
}
