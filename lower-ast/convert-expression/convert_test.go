package convert_expression

import (
	"testing"

	ast "xex/ast"
	ir_operator "xex/ast/operator"
	lower_ast "xex/lower-ast"
)

func checkSuffixString(t *testing.T, expression ast.ExpressionNode, expect string) {
	t.Helper()
	seq := ConvertExpression(expression, func(fn ast.CreateEnclosureExpression) lower_ast.FunctionProto {
		return lower_ast.NewFuncProto(nil, nil, nil, nil, nil, true)
	})
	if seq.IdentString(0) != expect {
		t.Fatalf("seq.IdentString() wrong.\ngot=%v\nwant=%v", seq.IdentString(0), expect)
	}
}

func TestConvertComplexExpressionString(t *testing.T) {
	expression := ast.CallExpression{
		Function: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "f"},
			Scope:      ast.IdentifierScopeFree,
		},
		Arguments: []ast.ExpressionNode{
			ast.ListLiteral{
				Elements: []ast.ExpressionNode{
					ast.Literial{Value: 1},
					ast.InfixExpression{
						Operator: ir_operator.PLUS,
						Left: ast.IdentifierExpression{
							Identifier: ast.Identifier{Value: "x"},
							Scope:      ast.IdentifierScopeLocal,
						},
						Right: ast.Literial{Value: 2},
					},
				},
			},
			ast.AssignmentExpression{
				Left: ast.LeftSetIndex{
					CanSetIndex: ast.IdentifierExpression{
						Identifier: ast.Identifier{Value: "arr"},
						Scope:      ast.IdentifierScopeLocal,
					},
					Index: ast.Literial{Value: 0},
				},
				Right: ast.PrefixExpression{
					Operator: ir_operator.MINUS,
					Right:    ast.Literial{Value: 3},
				},
			},
		},
	}

	expect := "" +
		"(identifier,free,f),(literal,1),(identifier,local,x),(literal,2),(infixExp,+),(list,2)," +
		"(identifier,local,arr),(literal,0),(literal,-3),(setIndex),(call,2)"

	checkSuffixString(t, expression, expect)
}

func TestConvertMapYieldAndClosureString(t *testing.T) {
	expression := ast.MapLiteral{
		Pairs: [][2]ast.ExpressionNode{
			{
				ast.Literial{Value: "k"},
				ast.YieldExpression{ReasonValue: ast.Literial{Value: "tick"}},
			},
			{
				ast.Literial{Value: "fn"},
				ast.CreateEnclosureExpression{
					ParameterSymbols: []ast.Identifier{{Value: "x"}},
				},
			},
		},
	}

	expect := "" +
		"(literal,k),(literal,tick),(yield),(literal,fn),(createProto,15994500279438875875),(map,2)"

	checkSuffixString(t, expression, expect)
}

func TestConvertNilComparisonString(t *testing.T) {
	checkSuffixString(t, ast.InfixExpression{
		Operator: ir_operator.EQ,
		Left: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "x"},
			Scope:      ast.IdentifierScopeLocal,
		},
		Right: ast.Literial{Value: nil},
	}, "(identifier,local,x),(literal,<nil>),(infixExp,==)")

	checkSuffixString(t, ast.InfixExpression{
		Operator: ir_operator.NOT_EQ,
		Left: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "x"},
			Scope:      ast.IdentifierScopeLocal,
		},
		Right: ast.Literial{Value: nil},
	}, "(identifier,local,x),(literal,<nil>),(infixExp,!=)")
}

func TestConvertBoolComparisonString(t *testing.T) {
	checkSuffixString(t, ast.InfixExpression{
		Operator: ir_operator.EQ,
		Left: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "x"},
			Scope:      ast.IdentifierScopeLocal,
		},
		Right: ast.Literial{Value: true},
	}, "(identifier,local,x),(literal,true),(infixExp,==)")

	checkSuffixString(t, ast.InfixExpression{
		Operator: ir_operator.NOT_EQ,
		Left: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "x"},
			Scope:      ast.IdentifierScopeLocal,
		},
		Right: ast.Literial{Value: false},
	}, "(identifier,local,x),(literal,false),(infixExp,!=)")

	checkSuffixString(t, ast.InfixExpression{
		Operator: ir_operator.EQ,
		Left: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "x"},
			Scope:      ast.IdentifierScopeLocal,
		},
		Right: ast.Literial{Value: false},
	}, "(identifier,local,x),(literal,false),(infixExp,==)")

	checkSuffixString(t, ast.InfixExpression{
		Operator: ir_operator.NOT_EQ,
		Left: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "x"},
			Scope:      ast.IdentifierScopeLocal,
		},
		Right: ast.Literial{Value: true},
	}, "(identifier,local,x),(literal,true),(infixExp,!=)")
}

func TestConvertNumberAndStringComparisonString(t *testing.T) {
	checkSuffixString(t, ast.InfixExpression{
		Operator: ir_operator.EQ,
		Left: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "x"},
			Scope:      ast.IdentifierScopeLocal,
		},
		Right: ast.Literial{Value: 7},
	}, "(identifier,local,x),(literal,7),(infixExp,==)")

	checkSuffixString(t, ast.InfixExpression{
		Operator: ir_operator.NOT_EQ,
		Left: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "x"},
			Scope:      ast.IdentifierScopeLocal,
		},
		Right: ast.Literial{Value: 9},
	}, "(identifier,local,x),(literal,9),(infixExp,!=)")

	checkSuffixString(t, ast.InfixExpression{
		Operator: ir_operator.EQ,
		Left: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "x"},
			Scope:      ast.IdentifierScopeLocal,
		},
		Right: ast.Literial{Value: "neo"},
	}, "(identifier,local,x),(literal,neo),(infixExp,==)")

	checkSuffixString(t, ast.InfixExpression{
		Operator: ir_operator.NOT_EQ,
		Left: ast.IdentifierExpression{
			Identifier: ast.Identifier{Value: "x"},
			Scope:      ast.IdentifierScopeLocal,
		},
		Right: ast.Literial{Value: "xex"},
	}, "(identifier,local,x),(literal,xex),(infixExp,!=)")
}

func TestConvertAssignmentAttributeAndIdentifierString(t *testing.T) {
	expression := ast.ListLiteral{
		Elements: []ast.ExpressionNode{
			ast.AssignmentExpression{
				Left: ast.LeftIdentifier{
					Identifier: ast.Identifier{Value: "x"},
					Scope:      ast.IdentifierScopeLocal,
				},
				Right: ast.Literial{Value: 1},
			},
			ast.AssignmentExpression{
				Left: ast.LeftSetAttribute{
					CanSetAttribute: ast.IdentifierExpression{
						Identifier: ast.Identifier{Value: "obj"},
						Scope:      ast.IdentifierScopeLocal,
					},
					Attribute: "name",
				},
				Right: ast.Literial{Value: "neo"},
			},
		},
	}

	expect := "" +
		"(literal,1),(setIdentifier,local,x),(identifier,local,obj),(literal,neo),(setAttr,name),(list,2)"

	checkSuffixString(t, expression, expect)
}

func TestConvertExpressionsKeepAllKeepsAllValues(t *testing.T) {
	seq := ConvertExpressionsKeepAll([]ast.ExpressionNode{
		ast.Literial{Value: 1},
		ast.Literial{Value: 2},
		ast.Literial{Value: 3},
	}, func(fn ast.CreateEnclosureExpression) lower_ast.FunctionProto {
		return lower_ast.NewFuncProto(nil, nil, nil, nil, nil, true)
	})

	if got := seq.IdentString(0); got != "(literal,1),(literal,2),(literal,3)" {
		t.Fatalf("ConvertExpressionsKeepAll() = %q", got)
	}
}

func TestConvertExpressionsKeepLastDropsPreviousValues(t *testing.T) {
	seq := ConvertExpressionsKeepLast([]ast.ExpressionNode{
		ast.Literial{Value: 1},
		ast.Literial{Value: 2},
		ast.Literial{Value: 3},
	}, func(fn ast.CreateEnclosureExpression) lower_ast.FunctionProto {
		return lower_ast.NewFuncProto(nil, nil, nil, nil, nil, true)
	})

	if got := seq.IdentString(0); got != "(literal,1)\n(literal,2)\n(literal,3)" {
		t.Fatalf("ConvertExpressionsKeepLast() = %q", got)
	}
}

func TestConvertExpressionPreservesLiteralTypes(t *testing.T) {
	seq := ConvertExpression(ast.Literial{Value: 7}, func(fn ast.CreateEnclosureExpression) lower_ast.FunctionProto {
		return lower_ast.NewFuncProto(nil, nil, nil, nil, nil, true)
	})
	literal, ok := seq[0].(lower_ast.LiterialExpression)
	if !ok {
		t.Fatalf("ConvertExpression(int literal) op = %T", seq[0])
	}
	if _, ok := literal.Value.(int); !ok {
		t.Fatalf("int literal type = %T, want int", literal.Value)
	}

	seq = ConvertExpression(ast.Literial{Value: float32(1.5)}, func(fn ast.CreateEnclosureExpression) lower_ast.FunctionProto {
		return lower_ast.NewFuncProto(nil, nil, nil, nil, nil, true)
	})
	literal, ok = seq[0].(lower_ast.LiterialExpression)
	if !ok {
		t.Fatalf("ConvertExpression(float32 literal) op = %T", seq[0])
	}
	if _, ok := literal.Value.(float32); !ok {
		t.Fatalf("float32 literal type = %T, want float32", literal.Value)
	}
}

func TestConvertExpressionSimplifiesLiteralOps(t *testing.T) {
	seq := ConvertExpression(ast.ListLiteral{
		Elements: []ast.ExpressionNode{
			ast.PrefixExpression{
				Operator: ir_operator.MINUS,
				Right:    ast.Literial{Value: 3},
			},
			ast.InfixExpression{
				Operator: ir_operator.PLUS,
				Left:     ast.Literial{Value: 4},
				Right:    ast.Literial{Value: 5},
			},
		},
	}, func(fn ast.CreateEnclosureExpression) lower_ast.FunctionProto {
		return lower_ast.NewFuncProto(nil, nil, nil, nil, nil, true)
	})

	if got := seq.IdentString(0); got != "(literal,-3),(literal,9),(list,2)" {
		t.Fatalf("ConvertExpression() simplified seq = %q", got)
	}
}

// func TestConvertExpressionPanicsOnUnsupportedLiteralType(t *testing.T) {
// 	defer func() {
// 		if recover() == nil {
// 			t.Fatal("ConvertExpression() did not panic for unsupported literal")
// 		}
// 	}()

// 	ConvertExpression(ast.Literial{Value: struct{ Name string }{Name: "bad"}}, func(fn ast.CreateEnclosureExpression) lower_ast.FunctionProto {
// 		fp := lower_ast.FunctionProto{}
// 		fp.HashValue = fp.RecomputeHash()
// 		return fp
// 	})
// }
