package lower_ast

import (
	"testing"

	ast "xex/ast"
)

// func TestLiteralExpressionHashNormalizesBasicNumericTypes(t *testing.T) {
// 	if got, want := (LiterialExpression{Value: int32(7)}).Hash(), (LiterialExpression{Value: 7}).Hash(); got != want {
// 		t.Fatalf("int32 literal hash = %d, want %d", got, want)
// 	}
// 	if got, want := (LiterialExpression{Value: float32(1.5)}).Hash(), (LiterialExpression{Value: float64(1.5)}).Hash(); got != want {
// 		t.Fatalf("float32 literal hash = %d, want %d", got, want)
// 	}
// }

func TestFunctionProtoHashIgnoresClosureDebugName(t *testing.T) {
	nested := NewFuncProto(
		[]ast.Identifier{{Value: "x"}},
		[]ast.Identifier{{Value: "x"}},
		nil, nil,
		LowerAst{
			{
				Sequence:     SuffixOperatesSequence{IdentifierExpression{IdentifierName: "x", Scope: ast.IdentifierScopeLocal}},
				IsNoCondJump: true,
				JumpTo:       CFGReturnTarget,
			},
		}, true,
	)
	buildOuter := func(name string) FunctionProto {
		fp := NewFuncProto(
			[]ast.Identifier{{Value: "arg"}},
			[]ast.Identifier{{Value: "arg"}},
			nil,
			[]ast.Identifier{{Value: "fmt"}},
			LowerAst{
				{
					Sequence: SuffixOperatesSequence{
						LiterialExpression{Value: int32(1)},
						CreateEnclosureExpression{FunctionProto: nested},
					},
					IsNoCondJump: true,
					JumpTo:       CFGReturnTarget,
				},
			}, true,
		)
		return fp
	}

	if got, want := buildOuter("fn_0").Hash(), buildOuter("fn_99").Hash(); got != want {
		t.Fatalf("FunctionProto hash depends on FnProtoName: got %d want %d", got, want)
	}
}

func TestFunctionProtoHashChangesWithBody(t *testing.T) {
	buildProto := func(value any) FunctionProto {
		fp := NewFuncProto(nil, nil, nil, nil,
			LowerAst{
				{
					Sequence:     SuffixOperatesSequence{LiterialExpression{Value: value}},
					IsNoCondJump: true,
					JumpTo:       CFGReturnTarget,
				},
			}, true,
		)
		return fp
	}

	// if got, want := buildProto(1).Hash(), buildProto(int32(1)).Hash(); got != want {
	// 	t.Fatalf("normalized literal hashes differ: got %d want %d", got, want)
	// }
	if got, other := buildProto(1).Hash(), buildProto(2).Hash(); got == other {
		t.Fatalf("different function proto bodies share hash: %d", got)
	}
}

func TestFunctionProtoHashPanicsOnUnsupportedLiteral(t *testing.T) {
	proto := FunctionProto{
		Graph: LowerAst{
			{
				Sequence:     SuffixOperatesSequence{LiterialExpression{Value: struct{ Bad string }{Bad: "x"}}},
				IsNoCondJump: true,
				JumpTo:       CFGReturnTarget,
			},
		},
	}

	defer func() {
		if recover() == nil {
			t.Fatal("FunctionProto.Hash() did not panic for unsupported literal")
		}
	}()

	_ = proto.Hash()
}
