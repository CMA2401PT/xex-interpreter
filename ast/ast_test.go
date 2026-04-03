package ast

import (
	"testing"

	ir_operator "xex/ast/operator"
)

func CheckAstString(t *testing.T, rootFunction CreateEnclosureExpression, expectString string) {
	t.Helper()
	if rootFunction.String() != expectString {
		t.Errorf("program.String() wrong. \ngot=%v, \nwant=%v", rootFunction.String(), expectString)
	}
}

func TestFib(t *testing.T) {
	iv := Identifier{Value: "v"}
	fib := CreateEnclosureExpression{
		ParameterSymbols: []Identifier{iv},
		FreeSymbols:      []Identifier{Identifier{Value: "fib"}},
		LocalSymbols:     []Identifier{Identifier{Value: "v"}},
		Body: StatementsStatement{
			Statements: []StatementNode{
				IfStatement{
					ConditionAndConsequence: []ConditionAndConsequence{
						ConditionAndConsequence{
							Cond: InfixExpression{Operator: ir_operator.EQ, Left: IdentifierExpression{Identifier: iv}, Right: Literial{0}},
							Do:   ReturnStatement{Literial{0}},
						},
						ConditionAndConsequence{
							Cond: InfixExpression{Operator: ir_operator.EQ, Left: IdentifierExpression{Identifier: iv}, Right: Literial{1}},
							Do:   ReturnStatement{Literial{1}},
						},
					},
					Alternative: EmptyStatement{},
				},
				ReturnStatement{InfixExpression{Operator: ir_operator.PLUS,
					Left: CallExpression{
						Function: IdentifierExpression{Identifier: Identifier{Value: "fib"}}, Arguments: []ExpressionNode{InfixExpression{Operator: ir_operator.MINUS, Left: IdentifierExpression{Identifier: iv}, Right: Literial{1}}},
					}, Right: CallExpression{
						Function: IdentifierExpression{Identifier: Identifier{Value: "fib"}}, Arguments: []ExpressionNode{InfixExpression{Operator: ir_operator.MINUS, Left: IdentifierExpression{Identifier: iv}, Right: Literial{2}}},
					}}},
			},
		},
	}
	ast := CreateEnclosureExpression{
		LocalSymbols: []Identifier{
			Identifier{Value: "fib"},
		},
		Body: ExpressionStatement{
			Expression: AssignmentExpression{
				Left:  LeftIdentifier{Identifier: Identifier{Value: "fib"}},
				Right: fib,
			},
		},
	}
	expect := "" +
		`function(){
	imports ;
	frees ;
	locals fib;
	fib = function(v){
		imports ;
		frees fib;
		locals v;
		if (v==0) {
			return 0;
		} else if (v==1) {
			return 1;
		};
		return (fib((v-1))+fib((v-2)));
	};
}`
	CheckAstString(t, ast, expect)
}

func TestCommonStructuresString(t *testing.T) {
	program := CreateEnclosureExpression{
		ParameterSymbols: []Identifier{{Value: "n"}},
		LocalSymbols:     []Identifier{{Value: "n"}, {Value: "arr"}, {Value: "dict"}, {Value: "i"}},
		ImportSymbols:    []Identifier{{Value: "fmt"}},
		Body: StatementsStatement{
			Statements: []StatementNode{
				ExpressionStatement{
					Expression: AssignmentExpression{
						Left: LeftIdentifier{Identifier: Identifier{Value: "arr"}, Scope: IdentifierScopeLocal},
						Right: ListLiteral{Elements: []ExpressionNode{
							Literial{Value: 1},
							Literial{Value: 2},
							IdentifierExpression{Identifier: Identifier{Value: "n"}},
						}},
					},
				},
				ExpressionStatement{
					Expression: AssignmentExpression{
						Left: LeftIdentifier{Identifier: Identifier{Value: "dict"}, Scope: IdentifierScopeLocal},
						Right: MapLiteral{Pairs: [][2]ExpressionNode{
							{Literial{Value: "name"}, Literial{Value: "neo"}},
							{Literial{Value: "count"}, IdentifierExpression{Identifier: Identifier{Value: "n"}}},
						}},
					},
				},
				ExpressionStatement{
					Expression: AssignmentExpression{
						Left: LeftIdentifier{Identifier: Identifier{Value: "i"}, Scope: IdentifierScopeLocal},
						Right: PrefixExpression{
							Operator: ir_operator.MINUS,
							Right:    Literial{Value: 1},
						},
					},
				},
				ExpressionStatement{
					Expression: AssignmentExpression{
						Left: LeftSetIndex{
							CanSetIndex: IdentifierExpression{Identifier: Identifier{Value: "arr"}},
							Index:       Literial{Value: 0},
						},
						Right: CallExpression{
							Function: IdentifierExpression{Identifier: Identifier{Value: "double"}},
							Arguments: []ExpressionNode{
								IdentifierExpression{Identifier: Identifier{Value: "n"}},
							},
						},
					},
				},
				ExpressionStatement{
					Expression: AssignmentExpression{
						Left: LeftSetAttribute{
							CanSetAttribute: IdentifierExpression{Identifier: Identifier{Value: "dict"}},
							Attribute:       "size",
						},
						Right: YieldExpression{
							ReasonValue: Literial{Value: "tick"},
						},
					},
				},
				IfStatement{
					ConditionAndConsequence: []ConditionAndConsequence{
						{
							Cond: InfixExpression{
								Operator: ir_operator.LT,
								Left:     IdentifierExpression{Identifier: Identifier{Value: "n"}},
								Right:    Literial{Value: 0},
							},
							Do: ReturnStatement{ReturnValue: Literial{Value: 0}},
						},
					},
					Alternative: ReturnStatement{ReturnValue: IdentifierExpression{Identifier: Identifier{Value: "n"}}},
				},
				LoopStatement{
					Condition: InfixExpression{
						Operator: ir_operator.LT,
						Left:     IdentifierExpression{Identifier: Identifier{Value: "i"}},
						Right:    IdentifierExpression{Identifier: Identifier{Value: "n"}},
					},
					AfterEachLoop: ExpressionStatement{AssignmentExpression{
						Left: LeftIdentifier{Identifier: Identifier{Value: "i"}, Scope: IdentifierScopeLocal},
						Right: InfixExpression{
							Operator: ir_operator.PLUS,
							Left:     IdentifierExpression{Identifier: Identifier{Value: "i"}},
							Right:    Literial{Value: 1},
						},
					}},
					LoopBody: StatementsStatement{
						Statements: []StatementNode{
							ContinueStatement{},
							BreakStatement{},
						},
					},
				},
			},
		},
	}

	expect := "" +
		`function(n){
	imports fmt;
	frees ;
	locals n,arr,dict,i;
	arr = [1,2,n];
	dict = {name:neo,count:n};
	i = (-1);
	arr[0] = double(n);
	dict.size = yield(tick);
	if (n<0) {
		return 0;
	} else {
		return n;
	};
	for (i<n);i = (i+1){
		continue;
		break;
	};
}`
	CheckAstString(t, program, expect)
}

func TestAstNodeMethodsCoverage(t *testing.T) {
	id := Identifier{Value: "x"}
	idExpr := IdentifierExpression{Identifier: id}
	literal := Literial{Value: 42}
	list := ListLiteral{Elements: []ExpressionNode{literal, idExpr}}
	hash := MapLiteral{Pairs: [][2]ExpressionNode{{literal, idExpr}}}
	yieldExpr := YieldExpression{ReasonValue: literal}
	prefix := PrefixExpression{Operator: ir_operator.BANG, Right: idExpr}
	infix := InfixExpression{Operator: ir_operator.ASTERISK, Left: literal, Right: idExpr}
	leftID := LeftIdentifier{Identifier: id, Scope: IdentifierScopeLocal}
	leftIndex := LeftSetIndex{CanSetIndex: idExpr, Index: literal}
	leftAttr := LeftSetAttribute{CanSetAttribute: idExpr, Attribute: "value"}
	assign := AssignmentExpression{Left: leftID, Right: literal}
	call := CallExpression{Function: idExpr, Arguments: []ExpressionNode{literal, idExpr}}
	empty := EmptyStatement{}
	ret := ReturnStatement{ReturnValue: literal}
	brk := BreakStatement{}
	cont := ContinueStatement{}
	exprStmt := ExpressionStatement{Expression: assign}
	block := StatementsStatement{Statements: []StatementNode{exprStmt, ret}}
	ifStmt := IfStatement{
		ConditionAndConsequence: []ConditionAndConsequence{{Cond: prefix, Do: brk}},
		Alternative:             empty,
	}
	loop := LoopStatement{Condition: prefix, AfterEachLoop: ExpressionStatement{assign}, LoopBody: cont}
	fn := CreateEnclosureExpression{ParameterSymbols: []Identifier{id}, Body: ret}

	if got := id.String(); got != "x" {
		t.Fatalf("Identifier.String() = %q", got)
	}
	if got := idExpr.String(); got != "x" {
		t.Fatalf("IdentifierExpression.String() = %q", got)
	}
	if got := literal.String(); got != "42" {
		t.Fatalf("Literial.String() = %q", got)
	}
	if got := list.String(); got != "[42,x]" {
		t.Fatalf("ListLiteral.String() = %q", got)
	}
	if got := hash.String(); got != "{42:x}" {
		t.Fatalf("MapLiteral.String() = %q", got)
	}
	if got := yieldExpr.String(); got != "yield(42)" {
		t.Fatalf("YieldExpression.String() = %q", got)
	}
	if got := prefix.String(); got != "(!x)" {
		t.Fatalf("PrefixExpression.String() = %q", got)
	}
	if got := infix.String(); got != "(42*x)" {
		t.Fatalf("InfixExpression.String() = %q", got)
	}
	if got := leftID.String(); got != "x" {
		t.Fatalf("LeftIdetifier.String() = %q", got)
	}
	if got := leftIndex.String(); got != "x[42]" {
		t.Fatalf("LeftSetIndex.String() = %q", got)
	}
	if got := leftAttr.String(); got != "x.value" {
		t.Fatalf("LeftSetAttribute.String() = %q", got)
	}
	if got := assign.String(); got != "x = 42" {
		t.Fatalf("AssignmentExpression.String() = %q", got)
	}
	if got := call.String(); got != "x(42,x)" {
		t.Fatalf("CallExpression.String() = %q", got)
	}
	if got := empty.String(); got != "" {
		t.Fatalf("EmptyStatement.String() = %q", got)
	}
	if got := empty.IdentString(3); got != "" {
		t.Fatalf("EmptyStatement.IdentString() = %q", got)
	}
	if got := ret.String(); got != "return 42;\n" {
		t.Fatalf("ReturnStatement.String() = %q", got)
	}
	if got := brk.String(); got != "break;\n" {
		t.Fatalf("BreakStatement.String() = %q", got)
	}
	if got := cont.String(); got != "continue;\n" {
		t.Fatalf("ContinueStatement.String() = %q", got)
	}
	if got := exprStmt.String(); got != "x = 42;\n" {
		t.Fatalf("ExpressionStatement.String() = %q", got)
	}
	if got := block.String(); got != "x = 42;\nreturn 42;\n" {
		t.Fatalf("StatementsStatement.String() = %q", got)
	}
	if got := ifStmt.String(); got != "if (!x) {\n\tbreak;\n};\n" {
		t.Fatalf("IfStatement.String() = %q", got)
	}
	if got := loop.String(); got != "for (!x);x = 42{\n\tcontinue;\n};\n" {
		t.Fatalf("LoopStatement.String() = %q", got)
	}
	if got := fn.NumArgs(); got != 1 {
		t.Fatalf("FunctionExpression.NumArgs() = %d", got)
	}

	assign.ExpressionNode()
	leftID.LeftValueNode()
	leftIndex.LeftValueNode()
	leftAttr.LeftValueNode()
	idExpr.ExpressionNode()
	literal.ExpressionNode()
	list.ExpressionNode()
	hash.ExpressionNode()
	yieldExpr.ExpressionNode()
	prefix.ExpressionNode()
	infix.ExpressionNode()
	call.ExpressionNode()
	fn.ExpressionNode()
	empty.StatementNode()
	ret.StatementNode()
	brk.StatementNode()
	cont.StatementNode()
	exprStmt.StatementNode()
	block.StatementNode()
	ifStmt.StatementNode()
	loop.StatementNode()
}

func TestSimplifyLiteralPrefixAndInfix(t *testing.T) {
	if got := (PrefixExpression{
		Operator: ir_operator.MINUS,
		Right:    Literial{Value: 3},
	}).Simplify(); got != (ExpressionNode)(Literial{Value: -3}) {
		t.Fatalf("PrefixExpression.Simplify() = %#v", got)
	}

	if got := (InfixExpression{
		Operator: ir_operator.PLUS,
		Left:     Literial{Value: 4},
		Right:    Literial{Value: 5},
	}).Simplify(); got != (ExpressionNode)(Literial{Value: 9}) {
		t.Fatalf("InfixExpression.Simplify() = %#v", got)
	}
}
