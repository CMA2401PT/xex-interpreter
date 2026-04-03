package convert_ast

import (
	"reflect"
	"testing"

	ast "xex/ast"
	ir_operator "xex/ast/operator"
	lower_ast "xex/lower-ast"
)

func testIdentifier(name string) ast.Identifier {
	return ast.Identifier{Value: name}
}

func testIdentifierExpression(name string, scope ast.IdentifierScope) ast.IdentifierExpression {
	return ast.IdentifierExpression{
		Identifier: testIdentifier(name),
		Scope:      scope,
	}
}

func TestConvertFnPreservesSymbolsAndLowersBody(t *testing.T) {
	fn := ast.CreateEnclosureExpression{
		ParameterSymbols: []ast.Identifier{testIdentifier("arg")},
		LocalSymbols:     []ast.Identifier{testIdentifier("arg"), testIdentifier("tmp")},
		FreeSymbols:      []ast.Identifier{testIdentifier("captured")},
		ImportSymbols:    []ast.Identifier{testIdentifier("fmt")},
		Body: ast.ReturnStatement{
			ReturnValue: testIdentifierExpression("arg", ast.IdentifierScopeLocal),
		},
	}

	proto := ConvertFn(fn)

	if proto.Hash() != 10693508910161154992 {
		t.Fatalf("ConvertFn() hash = %v, want %v", proto.Hash(), uint64(10693508910161154992))
	}
	if !reflect.DeepEqual(proto.ParameterSymbols, fn.ParameterSymbols) {
		t.Fatalf("ConvertFn() ParameterSymbols = %#v, want %#v", proto.ParameterSymbols, fn.ParameterSymbols)
	}
	if !reflect.DeepEqual(proto.LocalSymbols, fn.LocalSymbols) {
		t.Fatalf("ConvertFn() LocalSymbols = %#v, want %#v", proto.LocalSymbols, fn.LocalSymbols)
	}
	if !reflect.DeepEqual(proto.FreeSymbols, fn.FreeSymbols) {
		t.Fatalf("ConvertFn() FreeSymbols = %#v, want %#v", proto.FreeSymbols, fn.FreeSymbols)
	}
	if !reflect.DeepEqual(proto.ImportSymbols, fn.ImportSymbols) {
		t.Fatalf("ConvertFn() ImportSymbols = %#v, want %#v", proto.ImportSymbols, fn.ImportSymbols)
	}
	if got := proto.Graph[0].Sequence.IdentString(0); got != "(identifier,local,arg)" {
		t.Fatalf("ConvertFn() graph sequence = %q", got)
	}
	if !proto.Graph[0].IsNoCondJump || proto.Graph[0].JumpTo != lower_ast.CFGReturnTarget {
		t.Fatalf("ConvertFn() graph terminator = %+v", proto.Graph[0])
	}
}

func TestConvertAstCreateEnclosureReturnsLowerClosure(t *testing.T) {
	out := ConvertAst(ast.CreateEnclosureExpression{
		ParameterSymbols: []ast.Identifier{testIdentifier("x")},
		LocalSymbols:     []ast.Identifier{testIdentifier("x"), testIdentifier("inner")},
		Body: ast.ReturnStatement{
			ReturnValue: testIdentifierExpression("x", ast.IdentifierScopeLocal),
		},
	}).(lower_ast.CreateEnclosureExpression)

	if out.Hash() != 5218024282574867676 {
		t.Fatalf("ConvertAst(CreateEnclosureExpression).Hash = %v", out.Hash())
	}
	if !reflect.DeepEqual(out.FunctionProto.LocalSymbols, []ast.Identifier{testIdentifier("x"), testIdentifier("inner")}) {
		t.Fatalf("ConvertAst(CreateEnclosureExpression).LocalSymbols = %#v", out.FunctionProto.LocalSymbols)
	}
	if got := out.FunctionProto.Graph[0].Sequence.IdentString(0); got != "(identifier,local,x)" {
		t.Fatalf("ConvertAst(CreateEnclosureExpression).graph = %q", got)
	}
}

func TestConvertAstStatementLowersBlockSequenceAndNestedClosure(t *testing.T) {
	graph := ConvertAst(ast.StatementsStatement{
		Statements: []ast.StatementNode{
			ast.ExpressionStatement{Expression: ast.Literial{Value: int32(1)}},
			ast.ExpressionStatement{
				Expression: ast.CreateEnclosureExpression{
					ParameterSymbols: []ast.Identifier{testIdentifier("x")},
					LocalSymbols:     []ast.Identifier{testIdentifier("x"), testIdentifier("tmp")},
					Body: ast.ReturnStatement{
						ReturnValue: testIdentifierExpression("x", ast.IdentifierScopeLocal),
					},
				},
			},
		},
	}).(lower_ast.LowerAst)

	if len(graph) != 1 {
		t.Fatalf("ConvertAst(statement) blocks = %d, want 1", len(graph))
	}
	if got := graph[0].Sequence.IdentString(0); got != "(literal,1)\n(createProto,11883540583874823036)\n(literal,<nil>)" {
		t.Fatalf("ConvertAst(statement) sequence = %q", got)
	}
	createProto, ok := graph[0].Sequence[2].(lower_ast.CreateEnclosureExpression)
	if !ok {
		t.Fatalf("ConvertAst(statement) op[2] = %T, want CreateEnclosureExpression", graph[0].Sequence[2])
	}
	if !reflect.DeepEqual(createProto.FunctionProto.LocalSymbols, []ast.Identifier{testIdentifier("x"), testIdentifier("tmp")}) {
		t.Fatalf("nested closure LocalSymbols = %#v", createProto.FunctionProto.LocalSymbols)
	}
	if !graph[0].IsNoCondJump || graph[0].JumpTo != lower_ast.CFGReturnTarget {
		t.Fatalf("ConvertAst(statement) terminator = %+v", graph[0])
	}
}

func TestConvertAstExpressionDelegatesToExpressionConverter(t *testing.T) {
	sequence := ConvertAst(ast.InfixExpression{
		Operator: ir_operator.PLUS,
		Left:     ast.Literial{Value: int32(1)},
		Right:    testIdentifierExpression("x", ast.IdentifierScopeFree),
	}).(lower_ast.SuffixOperatesSequence)

	if got := sequence.IdentString(0); got != "(literal,1),(identifier,free,x),(infixExp,+)" {
		t.Fatalf("ConvertAst(expression) = %q", got)
	}
}

func TestConvertAstPanicsOnUnknownType(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("ConvertAst() did not panic for unknown AST type")
		}
	}()

	ConvertAst(struct{ Value string }{Value: "bad"})
}
