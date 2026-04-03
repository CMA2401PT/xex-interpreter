package convert_statement

import (
	"testing"
	ast "xex/ast"
)

func checkCFGString(t *testing.T, statement ast.StatementNode, expect string) {
	t.Helper()
	graph := StatementToCFGraph(statement)
	if graph.String() != expect {
		t.Fatalf("graph.String() wrong.\ngot=%v\nwant=%v", graph.String(), expect)
	}
}

func idExpr(name string) ast.IdentifierExpression {
	return ast.IdentifierExpression{Identifier: ast.Identifier{Value: name}}
}

func assignStmt(name string, right ast.ExpressionNode) ast.ExpressionStatement {
	return ast.ExpressionStatement{
		Expression: ast.AssignmentExpression{
			Left:  ast.LeftIdentifier{Identifier: ast.Identifier{Value: name}},
			Right: right,
		},
	}
}

func exprStmt(expression ast.ExpressionNode) ast.ExpressionStatement {
	return ast.ExpressionStatement{Expression: expression}
}

func TestStatementToCFGraphImplicitReturnString(t *testing.T) {
	statement := ast.StatementsStatement{
		Statements: []ast.StatementNode{
			assignStmt("x", ast.Literial{Value: int32(1)}),
		},
	}

	expect := "" +
		"0:\n" +
		"\tx = 1\n" +
		"\t<nil>\n" +
		"\treturn\n"

	checkCFGString(t, statement, expect)
}

func TestStatementToCFGraphElseIfString(t *testing.T) {
	statement := ast.IfStatement{
		ConditionAndConsequence: []ast.ConditionAndConsequence{
			{
				Cond: idExpr("cond1"),
				Do:   assignStmt("x", ast.Literial{Value: int32(1)}),
			},
			{
				Cond: idExpr("cond2"),
				Do: ast.ReturnStatement{
					ReturnValue: ast.Literial{Value: int32(2)},
				},
			},
		},
		Alternative: assignStmt("x", ast.Literial{Value: int32(3)}),
	}

	expect := "" +
		"0:\n" +
		"\tcond1\n" +
		"\tgoto 1 if True else 2\n" +
		"1:\n" +
		"\tx = 1\n" +
		"\tgoto 3\n" +
		"2:\n" +
		"\tcond2\n" +
		"\tgoto 4 if True else 5\n" +
		"3:\n" +
		"\t<nil>\n" +
		"\treturn\n" +
		"4:\n" +
		"\t2\n" +
		"\treturn\n" +
		"5:\n" +
		"\tx = 3\n" +
		"\tgoto 3\n"

	checkCFGString(t, statement, expect)
}

func TestStatementToCFGraphLoopWithIfString(t *testing.T) {
	statement := ast.LoopStatement{
		Condition:     idExpr("loopCond"),
		AfterEachLoop: exprStmt(idExpr("step")),
		LoopBody: ast.IfStatement{
			ConditionAndConsequence: []ast.ConditionAndConsequence{
				{
					Cond: idExpr("skip"),
					Do:   ast.ContinueStatement{},
				},
			},
			Alternative: ast.BreakStatement{},
		},
	}

	expect := "" +
		"0:\n" +
		"\tloopCond\n" +
		"\tgoto 1 if True else 3\n" +
		"1:\n" +
		"\tskip\n" +
		"\tgoto 4 if True else 5\n" +
		"2:\n" +
		"\tstep\n" +
		"\tloopCond\n" +
		"\tgoto 1 if True else 3\n" +
		"3:\n" +
		"\t<nil>\n" +
		"\treturn\n" +
		"4:\n" +
		"\tgoto 2\n" +
		"5:\n" +
		"\tgoto 3\n"

	checkCFGString(t, statement, expect)
}

func TestStatementToCFGraphNestedLoopString(t *testing.T) {
	statement := ast.LoopStatement{
		Condition:     idExpr("outerCond"),
		AfterEachLoop: exprStmt(idExpr("outerStep")),
		LoopBody: ast.LoopStatement{
			Condition:     idExpr("innerCond"),
			AfterEachLoop: exprStmt(idExpr("innerStep")),
			LoopBody:      ast.BreakStatement{},
		},
	}

	expect := "" +
		"0:\n" +
		"\touterCond\n" +
		"\tgoto 1 if True else 3\n" +
		"1:\n" +
		"\tinnerCond\n" +
		"\tgoto 4 if True else 6\n" +
		"2:\n" +
		"\touterStep\n" +
		"\touterCond\n" +
		"\tgoto 1 if True else 3\n" +
		"3:\n" +
		"\t<nil>\n" +
		"\treturn\n" +
		"4:\n" +
		"\tgoto 6\n" +
		"5:\n" +
		"\tinnerStep\n" +
		"\tinnerCond\n" +
		"\tgoto 4 if True else 6\n" +
		"6:\n" +
		"\tgoto 2\n"

	checkCFGString(t, statement, expect)
}

func TestStatementToCFGraphLoopBackEdgeDoesNotReenterPrelude(t *testing.T) {
	statement := ast.StatementsStatement{
		Statements: []ast.StatementNode{
			assignStmt("x", ast.Literial{Value: int32(0)}),
			ast.LoopStatement{
				Condition:     idExpr("cond"),
				AfterEachLoop: exprStmt(idExpr("step")),
				LoopBody:      exprStmt(idExpr("body")),
			},
			assignStmt("y", ast.Literial{Value: int32(1)}),
		},
	}

	expect := "" +
		"0:\n" +
		"\tx = 0\n" +
		"\tcond\n" +
		"\tgoto 1 if True else 3\n" +
		"1:\n" +
		"\tbody\n" +
		"\tgoto 2\n" +
		"2:\n" +
		"\tstep\n" +
		"\tcond\n" +
		"\tgoto 1 if True else 3\n" +
		"3:\n" +
		"\ty = 1\n" +
		"\t<nil>\n" +
		"\treturn\n"

	checkCFGString(t, statement, expect)
}

func TestStatementToCFGraphLoopLiteralTrueWithoutLatchString(t *testing.T) {
	statement := ast.LoopStatement{
		Condition:     ast.Literial{Value: true},
		AfterEachLoop: ast.EmptyStatement{},
		LoopBody: ast.IfStatement{
			ConditionAndConsequence: []ast.ConditionAndConsequence{
				{
					Cond: idExpr("stop"),
					Do:   ast.BreakStatement{},
				},
			},
			Alternative: ast.ContinueStatement{},
		},
	}

	expect := "" +
		"0:\n" +
		"\tgoto 1\n" +
		"1:\n" +
		"\tstop\n" +
		"\tgoto 3 if True else 4\n" +
		"2:\n" +
		"\t<nil>\n" +
		"\treturn\n" +
		"3:\n" +
		"\tgoto 2\n" +
		"4:\n" +
		"\tgoto 1\n"

	checkCFGString(t, statement, expect)
}

func TestStatementToCFGraphLoopPanicsWhenAfterEachChangesBlock(t *testing.T) {
	statement := ast.LoopStatement{
		Condition: idExpr("cond"),
		AfterEachLoop: ast.IfStatement{
			ConditionAndConsequence: []ast.ConditionAndConsequence{
				{
					Cond: idExpr("bad"),
					Do:   exprStmt(idExpr("step")),
				},
			},
			Alternative: ast.EmptyStatement{},
		},
		LoopBody: exprStmt(idExpr("body")),
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("StatementToCFGraph() did not panic")
		}
	}()

	StatementToCFGraph(statement)
}
