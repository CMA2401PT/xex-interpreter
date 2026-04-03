package compile

import (
	"maps"
	"slices"
	"testing"
	"time"
	ast "xex/ast"
	ir_operator "xex/ast/operator"
	"xex/async"
	"xex/engine"
	lower_ast "xex/lower-ast"
	convert_ast "xex/lower-ast/convert-ast"
	"xex/object"
)

func newLocalIdenitifer(name string) ast.IdentifierExpression {
	return ast.IdentifierExpression{Identifier: ast.Identifier{Value: name}, Scope: ast.IdentifierScopeLocal}
}

func newFreeIdentifier(name string) ast.IdentifierExpression {
	return ast.IdentifierExpression{Identifier: ast.Identifier{Value: name}, Scope: ast.IdentifierScopeFree}
}

func newLocalLeftIdentifier(name string) ast.LeftIdentifier {
	return ast.LeftIdentifier{
		Identifier: ast.Identifier{Value: name},
		Scope:      ast.IdentifierScopeLocal,
	}
}

// func newLocalCall(name string, args ...ast.ExpressionNode) ast.CallExpression {
// 	return ast.CallExpression{
// 		Function:  newLocalIdenitifer(name),
// 		Arguments: args,
// 	}
// }

func newLocalAssignStatement(name string, right ast.ExpressionNode) ast.ExpressionStatement {
	return ast.ExpressionStatement{
		Expression: ast.AssignmentExpression{
			Left:  newLocalLeftIdentifier(name),
			Right: right,
		},
	}
}

// func compileWithLocalValues(localValues map[string]any, last any) (any, engine.Env) {
// 	return compileWithLocalValuesAllowAsync(localValues, last, true)
// }

// func compileWithLocalValuesAllowAsync(localValues map[string]any, last any, allowAsync bool) (any, engine.Env) {
// 	names := slices.Collect(maps.Keys(localValues))
// 	lookup := genIdentifierScopsLookup([]IdentiferNamesWithScope{
// 		IdentiferNamesWithScope{
// 			symbolNames: names, scopeName: ast.IdentifierScopeLocal,
// 		},
// 	})
// 	locals := make([]object.CellOrVal, len(localValues))
// 	for n, v := range localValues {
// 		locals[lookup[lower_ast.IdentifierExpression{IdentifierName: n, Scope: ast.IdentifierScopeLocal}]] = object.CellOrVal{
// 			Value: v,
// 		}
// 	}

// 	hops := Compile(last, lookup, nil, allowAsync, map[string]functionProto{})
// 	return hops, engine.Env{
// 		GlobalsAndImport: nil, LocalAndFreeVars: locals,
// 	}
// }

// func compileExpressionWithLocalValues(
// 	t *testing.T,
// 	localValues map[string]any,
// 	ast ast.ExpressionNode,
// ) (HybridOperationsAndAuxSlotSize, engine.Env) {
// 	return compileExpressionWithLocalValuesAllowAsync(t, localValues, ast, true)
// }

// func compileExpressionWithLocalValuesAllowAsync(
// 	t *testing.T,
// 	localValues map[string]any,
// 	ast ast.ExpressionNode,
// 	allowAsync bool,
// ) (HybridOperationsAndAuxSlotSize, engine.Env) {
// 	t.Helper()
// 	hopsi, env := compileWithLocalValuesAllowAsync(localValues, convert_ast.ConvertAst(ast), allowAsync)
// 	hops, ok := hopsi.(HybridOperationsAndAuxSlotSize)
// 	if !ok {
// 		t.Fatalf("CompileAst() returned %T, want hybridOperationsAndAuxSlotSize", hopsi)
// 	}
// 	return hops, env
// }

// func compileStatementWithLocalValues(
// 	t *testing.T,
// 	localValues map[string]any,
// 	ast ast.StatementNode,
// ) (HybridOperationsAndAuxSlotSize, engine.Env) {
// 	return compileStatementWithLocalValuesAllowAsync(t, localValues, ast, true)
// }

// func compileStatementWithLocalValuesAllowAsync(
// 	t *testing.T,
// 	localValues map[string]any,
// 	ast ast.StatementNode,
// 	allowAsync bool,
// ) (HybridOperationsAndAuxSlotSize, engine.Env) {
// 	t.Helper()
// 	hopsi, env := compileWithLocalValuesAllowAsync(localValues, convert_ast.ConvertAst(ast), allowAsync)
// 	hops, ok := hopsi.(HybridOperationsAndAuxSlotSize)
// 	if !ok {
// 		t.Fatalf("CompileAst() returned %T, want HybridOperationsAndAuxSlotSize", hopsi)
// 	}
// 	return hops, env
// }

func compileRootEnclosure(
	t *testing.T,
	fn ast.CreateEnclosureExpression,
	globals map[string]object.Box,
) (*engine.Enclosure, FunctionProto, Lookup) {
	t.Helper()
	proto := convert_ast.ConvertFn(fn)
	globalNames := slices.Collect(maps.Keys(globals))
	fp, globalLookup := CompileFnProto(proto, globalNames, nil)
	globalValues := make([]object.Box, len(globalLookup))
	for name, value := range globals {
		slot := globalLookup[lower_ast.IdentifierExpression{
			IdentifierName: name,
			Scope:          ast.IdentifierScopeGlobal,
		}]
		globalValues[slot] = value
	}
	return fp.ToEnclosure(globalValues, nil), fp, globalLookup
}

// func callObjectCallable(t *testing.T, callable object.Object, args ...object.Object) object.Object {
// 	t.Helper()
// 	fn := object.GetCallable(callable)
// 	if fn == nil {
// 		t.Fatalf("callable is nil for %T", callable)
// 	}
// 	return fn(args)
// }

// func buildInnerLookupForProto(proto lower_ast.FunctionProto) Lookup {
// 	localSymbolsNames := make([]string, 0, len(proto.LocalSymbols))
// 	seen := map[string]struct{}{}
// 	for _, sym := range proto.ParameterSymbols {
// 		localSymbolsNames = append(localSymbolsNames, sym.Value)
// 		seen[sym.Value] = struct{}{}
// 	}
// 	for _, sym := range proto.LocalSymbols {
// 		if _, ok := seen[sym.Value]; ok {
// 			continue
// 		}
// 		localSymbolsNames = append(localSymbolsNames, sym.Value)
// 	}
// 	freeSymbolsNames := make([]string, 0, len(proto.FreeSymbols))
// 	for _, sym := range proto.FreeSymbols {
// 		freeSymbolsNames = append(freeSymbolsNames, sym.Value)
// 	}
// 	return genIdentifierScopsLookup([]IdentiferNamesWithScope{
// 		{symbolNames: localSymbolsNames, scopeName: ast.IdentifierScopeLocal, noSort: true},
// 		{symbolNames: freeSymbolsNames, scopeName: ast.IdentifierScopeFree, noSort: true},
// 	})
// }

// func runHybridOperations(t *testing.T, hops HybridOperationsAndAuxSlotSize, env engine.Env) engine.Env {
// 	t.Helper()
// 	return runHybridOperationsWithYield(t, hops, env, nil)
// }

// func runHybridOperationsWithYield(
// 	t *testing.T,
// 	hops HybridOperationsAndAuxSlotSize,
// 	env engine.Env,
// 	onYield func(slot uint8, reason any) any,
// ) engine.Env {
// 	t.Helper()
// 	env.Vars = make([]object.Object, hops.auxSlotSize)
// 	for _, op := range hops.operations {
// 		switch op.OperationType {
// 		default:
// 			t.Fatalf("unexpected operation type %v", op.OperationType)
// 		case engine.OpTypeGraphAndKeep:
// 			env.Vars[op.ResultSlot] = op.Graph(env)
// 		case engine.OpTypeGraphAndDrop:
// 			op.Graph(env)
// 		case engine.OpTypeCall:
// 			fn := object.GetCallable(env.Vars[op.FnSlot])
// 			env.Vars[op.FnSlot] = fn(env.Vars[op.ArgsStart:op.ArgsEnd])
// 		case engine.OpTypeYield:
// 			if onYield == nil {
// 				t.Fatalf("unexpected yield on slot %d with reason %v", op.ResultSlot, env.Vars[op.ResultSlot])
// 			}
// 			env.Vars[op.ResultSlot] = onYield(op.ResultSlot, env.Vars[op.ResultSlot])
// 		}
// 	}
// 	return env
// }

// func runCompiledLowerAst(t *testing.T, hops HybridOperationsAndAuxSlotSize, env engine.Env) (any, engine.Env) {
// 	t.Helper()
// 	env.Vars = make([]any, hops.auxSlotSize)
// 	pc := 0
// 	for steps := 0; steps < 10000; steps++ {
// 		if pc < 0 || pc >= len(hops.operations) {
// 			t.Fatalf("pc out of range: pc=%d len=%d", pc, len(hops.operations))
// 		}
// 		op := hops.operations[pc]
// 		switch op.OperationType {
// 		default:
// 			t.Fatalf("unexpected operation type %v at pc=%d", op.OperationType, pc)
// 		case engine.OpTypeGraphAndKeep:
// 			env.Vars[op.ResultSlot] = op.Graph(env)
// 			pc++
// 		case engine.OpTypeGraphAndDrop:
// 			op.Graph(env)
// 			pc++
// 		case engine.OpTypeCall:
// 			fn, ok := env.Vars[op.FnSlot].(func(args []any) any)
// 			if !ok {
// 				t.Fatalf("slot %d is %T, want func([]any) any", op.FnSlot, env.Vars[op.FnSlot])
// 			}
// 			env.Vars[op.FnSlot] = fn(env.Vars[op.ArgsStart:op.ArgsEnd])
// 			pc++
// 		case engine.OpTypeYield:
// 			t.Fatalf("unexpected yield at pc=%d", pc)
// 		case engine.OpTypeJumpNoCond:
// 			ret := op.Graph(env)
// 			if op.JumpTo == lower_ast.CFGReturnTarget {
// 				return ret, env
// 			}
// 			pc = int(op.JumpTo)
// 		case engine.OpTypeJumpTrue:
// 			condValue := op.Graph(env)
// 			cond := condValue.(bool)
// 			if cond {
// 				if op.JumpTo == lower_ast.CFGReturnTarget {
// 					panic(fmt.Errorf("unexpected conditional return target at pc=%d", pc))
// 				}
// 				pc = int(op.JumpTo)
// 			} else {
// 				pc += 1
// 			}
// 		}
// 	}
// 	t.Fatal("exceeded step limit while executing compiled lower ast")
// 	return nil, env
// }

// func describeHybridOperations(hops HybridOperationsAndAuxSlotSize) string {
// 	parts := make([]string, 0, len(hops.operations))
// 	for _, op := range hops.operations {
// 		switch op.OperationType {
// 		case engine.OpTypeGraphAndKeep:
// 			parts = append(parts, fmt.Sprintf("keep@%d", op.ResultSlot))
// 		case engine.OpTypeGraphAndDrop:
// 			parts = append(parts, "drop")
// 		case engine.OpTypeCall:
// 			parts = append(parts, fmt.Sprintf("call fn=%d args=%d:%d ->%d", op.FnSlot, op.ArgsStart, op.ArgsEnd, op.ResultSlot))
// 		case engine.OpTypeYield:
// 			parts = append(parts, fmt.Sprintf("yield@%d", op.ResultSlot))
// 		case engine.OpTypeJumpNoCond:
// 			if op.JumpTo == lower_ast.CFGReturnTarget {
// 				parts = append(parts, "jump->ret")
// 			} else {
// 				parts = append(parts, fmt.Sprintf("jump->%d", op.JumpTo))
// 			}
// 		case engine.OpTypeJumpTrue:
// 			if op.JumpTo == lower_ast.CFGReturnTarget {
// 				parts = append(parts, "jumpTrue->ret")
// 			} else {
// 				parts = append(parts, fmt.Sprintf("jumpTrue->%d", op.JumpTo))
// 			}
// 		default:
// 			parts = append(parts, fmt.Sprintf("unknown(%d)", op.OperationType))
// 		}
// 	}
// 	return strings.Join(parts, ", ")
// }

// func assertNoAsyncOps(t *testing.T, hops HybridOperationsAndAuxSlotSize) {
// 	t.Helper()
// 	for i, op := range hops.operations {
// 		if op.OperationType == engine.OpTypeCall || op.OperationType == engine.OpTypeYield {
// 			t.Fatalf("unexpected async op at index %d: %v", i, op.OperationType)
// 		}
// 	}
// }

// func TestIdentifierAndInfix(t *testing.T) {
// 	ast := ast.InfixExpression{
// 		Operator: ast.PLUS,
// 		Left:     newLocalIdenitifer("x1"),
// 		Right:    newLocalIdenitifer("x2"),
// 	}

// 	last := convert_ast.ConvertAst(ast)
// 	hopsi, env := compileWithLocalValues(map[string]any{"x1": 1, "x2": 2}, last)
// 	hops := hopsi.(HybridOperationsAndAuxSlotSize)
// 	env.Vars = make([]any, hops.auxSlotSize)
// 	for _, op := range hops.operations {
// 		switch op.OperationType {
// 		default:
// 			t.FailNow()
// 		case engine.OpTypeGraphAndKeep:
// 			env.Vars[op.ResultSlot] = op.Graph(env)
// 		case engine.OpTypeGraphAndDrop:
// 			op.Graph(env)
// 		}
// 	}
// 	if env.Vars[0] != 3 && len(env.Vars) != 1 {
// 		t.FailNow()
// 	}
// }

// func TestNilComparisonSpecialOps(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		operator   ast.Operator
// 		localValue any
// 		want       bool
// 	}{
// 		{name: "eq nil true", operator: ast.EQ, localValue: nil, want: true},
// 		{name: "eq nil false", operator: ast.EQ, localValue: 1, want: false},
// 		{name: "not eq nil false", operator: ast.NOT_EQ, localValue: nil, want: false},
// 		{name: "not eq nil true", operator: ast.NOT_EQ, localValue: 1, want: true},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			hops, env := compileExpressionWithLocalValues(t, map[string]any{"x": tt.localValue}, ast.InfixExpression{
// 				Operator: tt.operator,
// 				Left:     newLocalIdenitifer("x"),
// 				Right:    ast.Literial{Value: nil},
// 			})
// 			assertNoAsyncOps(t, hops)

// 			if got := describeHybridOperations(hops); got != "keep@0" {
// 				t.Fatalf("hybrid operations = %q, want %q", got, "keep@0")
// 			}

// 			env = runHybridOperations(t, hops, env)
// 			if got := env.Vars[0]; got != tt.want {
// 				t.Fatalf("result = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestBoolComparisonSpecialOps(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		operator   ast.Operator
// 		rightValue bool
// 		localValue any
// 		want       bool
// 	}{
// 		{name: "eq true true", operator: ast.EQ, rightValue: true, localValue: true, want: true},
// 		{name: "eq true false", operator: ast.EQ, rightValue: true, localValue: false, want: false},
// 		{name: "not eq false true", operator: ast.NOT_EQ, rightValue: false, localValue: true, want: true},
// 		{name: "eq false true", operator: ast.EQ, rightValue: false, localValue: false, want: true},
// 		{name: "not eq true true", operator: ast.NOT_EQ, rightValue: true, localValue: false, want: true},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			hops, env := compileExpressionWithLocalValues(t, map[string]any{"x": tt.localValue}, ast.InfixExpression{
// 				Operator: tt.operator,
// 				Left:     newLocalIdenitifer("x"),
// 				Right:    ast.Literial{Value: tt.rightValue},
// 			})
// 			assertNoAsyncOps(t, hops)

// 			if got := describeHybridOperations(hops); got != "keep@0" {
// 				t.Fatalf("hybrid operations = %q, want %q", got, "keep@0")
// 			}

// 			env = runHybridOperations(t, hops, env)
// 			if got := env.Vars[0]; got != tt.want {
// 				t.Fatalf("result = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestBoolComparisonSpecialOpsTypeMismatch(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		operator   ast.Operator
// 		rightValue bool
// 		localValue any
// 	}{
// 		{name: "eq true int", operator: ast.EQ, rightValue: true, localValue: 1},
// 		{name: "eq false int", operator: ast.EQ, rightValue: false, localValue: 1},
// 		{name: "not eq true int", operator: ast.NOT_EQ, rightValue: true, localValue: 1},
// 		{name: "not eq false int", operator: ast.NOT_EQ, rightValue: false, localValue: 1},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			hops, env := compileExpressionWithLocalValues(t, map[string]any{"x": tt.localValue}, ast.InfixExpression{
// 				Operator: tt.operator,
// 				Left:     newLocalIdenitifer("x"),
// 				Right:    ast.Literial{Value: tt.rightValue},
// 			})

// 			defer func() {
// 				if r := recover(); r == nil {
// 					t.Fatal("expected panic for type mismatch")
// 				}
// 			}()
// 			runHybridOperations(t, hops, env)
// 		})
// 	}
// }

// func TestNumberAndStringComparisonSpecialOps(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		operator   ast.Operator
// 		rightValue any
// 		localValue any
// 		want       bool
// 	}{
// 		{name: "eq number true", operator: ast.EQ, rightValue: 7, localValue: 7, want: true},
// 		{name: "eq number false", operator: ast.EQ, rightValue: 7, localValue: 8, want: false},
// 		{name: "not eq number true", operator: ast.NOT_EQ, rightValue: 7, localValue: 8, want: true},
// 		{name: "eq string true", operator: ast.EQ, rightValue: "neo", localValue: "neo", want: true},
// 		{name: "eq string false", operator: ast.EQ, rightValue: "neo", localValue: "xex", want: false},
// 		{name: "not eq string true", operator: ast.NOT_EQ, rightValue: "neo", localValue: "xex", want: true},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			hops, env := compileExpressionWithLocalValues(t, map[string]any{"x": tt.localValue}, ast.InfixExpression{
// 				Operator: tt.operator,
// 				Left:     newLocalIdenitifer("x"),
// 				Right:    ast.Literial{Value: tt.rightValue},
// 			})
// 			assertNoAsyncOps(t, hops)

// 			if got := describeHybridOperations(hops); got != "keep@0" {
// 				t.Fatalf("hybrid operations = %q, want %q", got, "keep@0")
// 			}

// 			env = runHybridOperations(t, hops, env)
// 			if got := env.Vars[0]; got != tt.want {
// 				t.Fatalf("result = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestNumberAndStringComparisonSpecialOpsTypeMismatch(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		operator   ast.Operator
// 		rightValue any
// 		localValue any
// 	}{
// 		{name: "eq number string", operator: ast.EQ, rightValue: 7, localValue: "neo"},
// 		{name: "not eq number string", operator: ast.NOT_EQ, rightValue: 7, localValue: "neo"},
// 		{name: "eq string int", operator: ast.EQ, rightValue: "neo", localValue: 1},
// 		{name: "not eq string int", operator: ast.NOT_EQ, rightValue: "neo", localValue: 1},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			hops, env := compileExpressionWithLocalValues(t, map[string]any{"x": tt.localValue}, ast.InfixExpression{
// 				Operator: tt.operator,
// 				Left:     newLocalIdenitifer("x"),
// 				Right:    ast.Literial{Value: tt.rightValue},
// 			})

// 			defer func() {
// 				if r := recover(); r == nil {
// 					t.Fatal("expected panic for type mismatch")
// 				}
// 			}()
// 			runHybridOperations(t, hops, env)
// 		})
// 	}
// }

// func TestDrop(t *testing.T) {
// 	// 这里中间会生成一个 Drop 操作，检查是否能够正确处理 Drop 操作
// 	ast := ast.StatementsStatement{
// 		Statements: []ast.StatementNode{
// 			ast.ExpressionStatement{
// 				Expression: ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left:     newLocalIdenitifer("x1"),
// 					Right:    newLocalIdenitifer("x2"),
// 				},
// 			},
// 			ast.ReturnStatement{
// 				ReturnValue: ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left:     newLocalIdenitifer("x1"),
// 					Right:    newLocalIdenitifer("x2"),
// 				},
// 			},
// 		},
// 	}

// 	last := convert_ast.ConvertAst(ast).(lower_ast.LowerAst)
// 	if len(last) != 1 {
// 		t.FailNow()
// 	}
// 	seq := last[0].Sequence
// 	hopsi, env := compileWithLocalValues(map[string]any{"x1": 1, "x2": 2}, seq)
// 	hops := hopsi.(HybridOperationsAndAuxSlotSize)
// 	env.Vars = make([]any, hops.auxSlotSize)
// 	for _, op := range hops.operations {
// 		switch op.OperationType {
// 		default:
// 			t.FailNow()
// 		case engine.OpTypeGraphAndKeep:
// 			env.Vars[op.ResultSlot] = op.Graph(env)
// 		case engine.OpTypeGraphAndDrop:
// 			op.Graph(env)
// 		}
// 	}
// 	if env.Vars[0] != 3 && len(env.Vars) != 0 {
// 		t.FailNow()
// 	}
// }

// func TestAuxValues(t *testing.T) {
// 	// x5 + fn((x1+x2),(x3+x4))
// 	ast := ast.InfixExpression{
// 		Operator: ast.PLUS,
// 		Left:     newLocalIdenitifer("x5"),
// 		Right: ast.CallExpression{
// 			Function: newLocalIdenitifer("fn"),
// 			Arguments: []ast.ExpressionNode{
// 				ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left:     newLocalIdenitifer("x1"),
// 					Right:    newLocalIdenitifer("x2"),
// 				}, ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left:     newLocalIdenitifer("x3"),
// 					Right:    newLocalIdenitifer("x4"),
// 				},
// 			},
// 		},
// 	}

// 	last := convert_ast.ConvertAst(ast)
// 	// = 1000+3*300=1900
// 	hopsi, env := compileWithLocalValues(map[string]any{
// 		"x1": 1, "x2": 2,
// 		"x3": 100, "x4": 200,
// 		"x5": 1000,
// 		"fn": func(args []any) any {
// 			if len(args) != 2 {
// 				panic("must be 2 args")
// 			}
// 			return args[0].(int) * args[1].(int)
// 		},
// 	}, last)
// 	hops := hopsi.(HybridOperationsAndAuxSlotSize)
// 	// call 是特殊操作，所以不会变为子图的节点，因此其操作数应当落在辅助栈中
// 	// 第一个操作应当是取 x5的值的 Graph 并将值放入辅助栈 1
// 	// 第二个操作应当是取 x1+x2 的值的 Graph 并将值放入辅助栈 2
// 	//
// 	env.Vars = make([]any, hops.auxSlotSize)
// 	for _, op := range hops.operations {
// 		switch op.OperationType {
// 		default:
// 			t.FailNow()
// 		case engine.OpTypeGraphAndKeep:
// 			env.Vars[op.ResultSlot] = op.Graph(env)
// 		case engine.OpTypeGraphAndDrop:
// 			op.Graph(env)
// 		case engine.OpTypeCall:
// 			// 当运行到这一步时，其实函数和其值都已经在辅助栈上了，
// 			// 我们需要做的就是调用
// 			// 由于在这个test里我们知道它不是一个协程函数，
// 			// 而就是一个普通函数，所以我们直接执行
// 			fn := env.Vars[op.FnSlot].(func(args []any) any)
// 			env.Vars[op.FnSlot] = fn(env.Vars[op.ArgsStart:op.ArgsEnd])
// 		}
// 	}
// 	if env.Vars[0] != 1900 {
// 		t.FailNow()
// 	}
// }

// func TestCompileAstNestedCallBridgeLayout(t *testing.T) {
// 	// x0 + f(g(x1+x2) + h(x3+x4))
// 	ast := ast.InfixExpression{
// 		Operator: ast.PLUS,
// 		Left:     newLocalIdenitifer("x0"),
// 		Right: ast.CallExpression{
// 			Function: newLocalIdenitifer("f"),
// 			Arguments: []ast.ExpressionNode{
// 				ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left: ast.CallExpression{
// 						Function: newLocalIdenitifer("g"),
// 						Arguments: []ast.ExpressionNode{
// 							ast.InfixExpression{
// 								Operator: ast.PLUS,
// 								Left:     newLocalIdenitifer("x1"),
// 								Right:    newLocalIdenitifer("x2"),
// 							},
// 						},
// 					},
// 					Right: ast.CallExpression{
// 						Function: newLocalIdenitifer("h"),
// 						Arguments: []ast.ExpressionNode{
// 							ast.InfixExpression{
// 								Operator: ast.PLUS,
// 								Left:     newLocalIdenitifer("x3"),
// 								Right:    newLocalIdenitifer("x4"),
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	hops, env := compileExpressionWithLocalValues(t, map[string]any{
// 		"x0": 1, "x1": 2, "x2": 3,
// 		"x3": 4, "x4": 5,
// 		"f": func(args []any) any { return args[0].(int) + 7 },
// 		"g": func(args []any) any { return args[0].(int) * 10 },
// 		"h": func(args []any) any { return args[0].(int) * 100 },
// 	}, ast)

// 	if hops.auxSlotSize != 5 {
// 		t.Fatalf("auxSlotSize = %d, want 5", hops.auxSlotSize)
// 	}
// 	wantOps := strings.Join([]string{
// 		"keep@0",
// 		"keep@1",
// 		"keep@2",
// 		"keep@3",
// 		"call fn=2 args=3:4 ->2",
// 		"keep@3",
// 		"keep@4",
// 		"call fn=3 args=4:5 ->3",
// 		"keep@2",
// 		"call fn=1 args=2:3 ->1",
// 		"keep@0",
// 	}, ", ")
// 	if got := describeHybridOperations(hops); got != wantOps {
// 		t.Fatalf("compiled operations mismatch\ngot:  %s\nwant: %s", got, wantOps)
// 	}

// 	env = runHybridOperations(t, hops, env)
// 	if got := env.Vars[0]; got != 958 {
// 		t.Fatalf("final result = %v, want 958", got)
// 	}
// }

// func TestCompileAstNestedCallBridgesRemainStableAcrossMultipleLevels(t *testing.T) {
// 	callLog := make([]string, 0, 4)
// 	loggedUnary := func(name string, fn func(int) int) func(args []any) any {
// 		return func(args []any) any {
// 			if len(args) != 1 {
// 				t.Fatalf("%s expects 1 arg, got %d", name, len(args))
// 			}
// 			in := args[0].(int)
// 			out := fn(in)
// 			callLog = append(callLog, fmt.Sprintf("%s(%d)=%d", name, in, out))
// 			return out
// 		}
// 	}
// 	loggedBinary := func(name string, fn func(int, int) int) func(args []any) any {
// 		return func(args []any) any {
// 			if len(args) != 2 {
// 				t.Fatalf("%s expects 2 args, got %d", name, len(args))
// 			}
// 			left := args[0].(int)
// 			right := args[1].(int)
// 			out := fn(left, right)
// 			callLog = append(callLog, fmt.Sprintf("%s(%d,%d)=%d", name, left, right, out))
// 			return out
// 		}
// 	}

// 	// f1(x1 + f2(x2 + f3(x3+x4)), f4(x5+x6) + x7) + x8
// 	ast := ast.InfixExpression{
// 		Operator: ast.PLUS,
// 		Left: ast.CallExpression{
// 			Function: newLocalIdenitifer("f1"),
// 			Arguments: []ast.ExpressionNode{
// 				ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left:     newLocalIdenitifer("x1"),
// 					Right: ast.CallExpression{
// 						Function: newLocalIdenitifer("f2"),
// 						Arguments: []ast.ExpressionNode{
// 							ast.InfixExpression{
// 								Operator: ast.PLUS,
// 								Left:     newLocalIdenitifer("x2"),
// 								Right: ast.CallExpression{
// 									Function: newLocalIdenitifer("f3"),
// 									Arguments: []ast.ExpressionNode{
// 										ast.InfixExpression{
// 											Operator: ast.PLUS,
// 											Left:     newLocalIdenitifer("x3"),
// 											Right:    newLocalIdenitifer("x4"),
// 										},
// 									},
// 								},
// 							},
// 						},
// 					},
// 				},
// 				ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left: ast.CallExpression{
// 						Function: newLocalIdenitifer("f4"),
// 						Arguments: []ast.ExpressionNode{
// 							ast.InfixExpression{
// 								Operator: ast.PLUS,
// 								Left:     newLocalIdenitifer("x5"),
// 								Right:    newLocalIdenitifer("x6"),
// 							},
// 						},
// 					},
// 					Right: newLocalIdenitifer("x7"),
// 				},
// 			},
// 		},
// 		Right: newLocalIdenitifer("x8"),
// 	}

// 	hops, env := compileExpressionWithLocalValues(t, map[string]any{
// 		"x1": 1, "x2": 2, "x3": 3, "x4": 4,
// 		"x5": 5, "x6": 6, "x7": 7, "x8": 8,
// 		"f1": loggedBinary("f1", func(left, right int) int { return left*10 + right }),
// 		"f2": loggedUnary("f2", func(v int) int { return v * 2 }),
// 		"f3": loggedUnary("f3", func(v int) int { return v + 100 }),
// 		"f4": loggedUnary("f4", func(v int) int { return v + 1000 }),
// 	}, ast)

// 	env = runHybridOperations(t, hops, env)

// 	if got := env.Vars[0]; got != 3216 {
// 		t.Fatalf("final result = %v, want 3216", got)
// 	}
// 	wantLog := []string{
// 		"f3(7)=107",
// 		"f2(109)=218",
// 		"f4(11)=1011",
// 		"f1(219,1018)=3208",
// 	}
// 	if !slices.Equal(callLog, wantLog) {
// 		t.Fatalf("call order mismatch\ngot:  %v\nwant: %v", callLog, wantLog)
// 	}
// }

// func TestCompileAstYieldBridgeLayout(t *testing.T) {
// 	// x0 + yield(x1+x2)
// 	ast := ast.InfixExpression{
// 		Operator: ast.PLUS,
// 		Left:     newLocalIdenitifer("x0"),
// 		Right: ast.YieldExpression{
// 			ReasonValue: ast.InfixExpression{
// 				Operator: ast.PLUS,
// 				Left:     newLocalIdenitifer("x1"),
// 				Right:    newLocalIdenitifer("x2"),
// 			},
// 		},
// 	}

// 	hops, env := compileExpressionWithLocalValues(t, map[string]any{
// 		"x0": 10, "x1": 1, "x2": 2,
// 	}, ast)

// 	if hops.auxSlotSize != 2 {
// 		t.Fatalf("auxSlotSize = %d, want 2", hops.auxSlotSize)
// 	}
// 	wantOps := strings.Join([]string{
// 		"keep@0",
// 		"keep@1",
// 		"yield@1",
// 		"keep@0",
// 	}, ", ")
// 	if got := describeHybridOperations(hops); got != wantOps {
// 		t.Fatalf("compiled operations mismatch\ngot:  %s\nwant: %s", got, wantOps)
// 	}

// 	yieldReasons := make([]any, 0, 1)
// 	env = runHybridOperationsWithYield(t, hops, env, func(slot uint8, reason any) any {
// 		if slot != 1 {
// 			t.Fatalf("yield slot = %d, want 1", slot)
// 		}
// 		yieldReasons = append(yieldReasons, reason)
// 		return 100
// 	})
// 	if len(yieldReasons) != 1 || yieldReasons[0] != 3 {
// 		t.Fatalf("yield reasons = %v, want [3]", yieldReasons)
// 	}
// 	if got := env.Vars[0]; got != 110 {
// 		t.Fatalf("final result = %v, want 110", got)
// 	}
// }

// func TestCompileAstYieldAndCallBridgesRemainStableAcrossMultipleSuspends(t *testing.T) {
// 	yieldReasons := make([]any, 0, 2)
// 	callLog := make([]string, 0, 1)

// 	// x0 + fn(yield(x1+x2), x3 + yield(x4+x5))
// 	ast := ast.InfixExpression{
// 		Operator: ast.PLUS,
// 		Left:     newLocalIdenitifer("x0"),
// 		Right: ast.CallExpression{
// 			Function: newLocalIdenitifer("fn"),
// 			Arguments: []ast.ExpressionNode{
// 				ast.YieldExpression{
// 					ReasonValue: ast.InfixExpression{
// 						Operator: ast.PLUS,
// 						Left:     newLocalIdenitifer("x1"),
// 						Right:    newLocalIdenitifer("x2"),
// 					},
// 				},
// 				ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left:     newLocalIdenitifer("x3"),
// 					Right: ast.YieldExpression{
// 						ReasonValue: ast.InfixExpression{
// 							Operator: ast.PLUS,
// 							Left:     newLocalIdenitifer("x4"),
// 							Right:    newLocalIdenitifer("x5"),
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	hops, env := compileExpressionWithLocalValues(t, map[string]any{
// 		"x0": 1, "x1": 2, "x2": 3, "x3": 4, "x4": 5, "x5": 6,
// 		"fn": func(args []any) any {
// 			if len(args) != 2 {
// 				t.Fatalf("fn expects 2 args, got %d", len(args))
// 			}
// 			left := args[0].(int)
// 			right := args[1].(int)
// 			out := left*10 + right
// 			callLog = append(callLog, fmt.Sprintf("fn(%d,%d)=%d", left, right, out))
// 			return out
// 		},
// 	}, ast)

// 	wantOps := strings.Join([]string{
// 		"keep@0",
// 		"keep@1",
// 		"keep@2",
// 		"yield@2",
// 		"keep@3",
// 		"keep@4",
// 		"yield@4",
// 		"keep@3",
// 		"call fn=1 args=2:4 ->1",
// 		"keep@0",
// 	}, ", ")
// 	if got := describeHybridOperations(hops); got != wantOps {
// 		t.Fatalf("compiled operations mismatch\ngot:  %s\nwant: %s", got, wantOps)
// 	}

// 	resumeValues := []any{100, 200}
// 	env = runHybridOperationsWithYield(t, hops, env, func(slot uint8, reason any) any {
// 		yieldReasons = append(yieldReasons, fmt.Sprintf("slot%d:%v", slot, reason))
// 		if len(yieldReasons) > len(resumeValues) {
// 			t.Fatalf("unexpected extra yield: slot=%d reason=%v", slot, reason)
// 		}
// 		return resumeValues[len(yieldReasons)-1]
// 	})

// 	wantYieldReasons := []any{"slot2:5", "slot4:11"}
// 	if !slices.Equal(yieldReasons, wantYieldReasons) {
// 		t.Fatalf("yield sequence mismatch\ngot:  %v\nwant: %v", yieldReasons, wantYieldReasons)
// 	}
// 	wantCallLog := []string{"fn(100,204)=1204"}
// 	if !slices.Equal(callLog, wantCallLog) {
// 		t.Fatalf("call log mismatch\ngot:  %v\nwant: %v", callLog, wantCallLog)
// 	}
// 	if got := env.Vars[0]; got != 1205 {
// 		t.Fatalf("final result = %v, want 1205", got)
// 	}
// }

// func TestCompileAstAlreadyFallbackedValuesAreNotLoweredTwice(t *testing.T) {
// 	// x0 + fn(yield(x1+x2), yield(x3+x4))
// 	// 这里 x0、fn 以及第一次 yield 的恢复值都会跨过后续特殊操作继续存活。
// 	// 它们一旦已经 bridge 到 aux 槽，后面的 yield/call 就不应该再生成新的 GraphEval。
// 	ast := ast.InfixExpression{
// 		Operator: ast.PLUS,
// 		Left:     newLocalIdenitifer("x0"),
// 		Right: ast.CallExpression{
// 			Function: newLocalIdenitifer("fn"),
// 			Arguments: []ast.ExpressionNode{
// 				ast.YieldExpression{
// 					ReasonValue: ast.InfixExpression{
// 						Operator: ast.PLUS,
// 						Left:     newLocalIdenitifer("x1"),
// 						Right:    newLocalIdenitifer("x2"),
// 					},
// 				},
// 				ast.YieldExpression{
// 					ReasonValue: ast.InfixExpression{
// 						Operator: ast.PLUS,
// 						Left:     newLocalIdenitifer("x3"),
// 						Right:    newLocalIdenitifer("x4"),
// 					},
// 				},
// 			},
// 		},
// 	}

// 	callLog := make([]string, 0, 1)
// 	hops, env := compileExpressionWithLocalValues(t, map[string]any{
// 		"x0": 1, "x1": 2, "x2": 3, "x3": 4, "x4": 5,
// 		"fn": func(args []any) any {
// 			if len(args) != 2 {
// 				t.Fatalf("fn expects 2 args, got %d", len(args))
// 			}
// 			left := args[0].(int)
// 			right := args[1].(int)
// 			out := left*100 + right
// 			callLog = append(callLog, fmt.Sprintf("fn(%d,%d)=%d", left, right, out))
// 			return out
// 		},
// 	}, ast)

// 	wantOps := strings.Join([]string{
// 		"keep@0",
// 		"keep@1",
// 		"keep@2",
// 		"yield@2",
// 		"keep@3",
// 		"yield@3",
// 		"call fn=1 args=2:4 ->1",
// 		"keep@0",
// 	}, ", ")
// 	if got := describeHybridOperations(hops); got != wantOps {
// 		t.Fatalf("compiled operations mismatch\ngot:  %s\nwant: %s", got, wantOps)
// 	}

// 	yieldReasons := make([]string, 0, 2)
// 	env = runHybridOperationsWithYield(t, hops, env, func(slot uint8, reason any) any {
// 		yieldReasons = append(yieldReasons, fmt.Sprintf("slot%d:%v", slot, reason))
// 		switch len(yieldReasons) {
// 		case 1:
// 			return 10
// 		case 2:
// 			return 20
// 		default:
// 			t.Fatalf("unexpected extra yield: slot=%d reason=%v", slot, reason)
// 			return nil
// 		}
// 	})

// 	wantYieldReasons := []string{"slot2:5", "slot3:9"}
// 	if !slices.Equal(yieldReasons, wantYieldReasons) {
// 		t.Fatalf("yield sequence mismatch\ngot:  %v\nwant: %v", yieldReasons, wantYieldReasons)
// 	}
// 	wantCallLog := []string{"fn(10,20)=1020"}
// 	if !slices.Equal(callLog, wantCallLog) {
// 		t.Fatalf("call log mismatch\ngot:  %v\nwant: %v", callLog, wantCallLog)
// 	}
// 	if got := env.Vars[0]; got != 1021 {
// 		t.Fatalf("final result = %v, want 1021", got)
// 	}
// }

// func TestCompileLowerAstPrefersFalseFallthrough(t *testing.T) {
// 	graph := lower_ast.LowerAst{
// 		{
// 			Sequence: lower_ast.SuffixOperatesSequence{lower_ast.IdentifierExpression{
// 				IdentifierName: "flag",
// 				Scope:          ast.IdentifierScopeLocal,
// 			}},
// 			IsNoCondJump: false,
// 			JumpTo:       2,
// 			Aux:          1,
// 		},
// 		{
// 			Sequence:     lower_ast.SuffixOperatesSequence{lower_ast.LiterialExpression{Value: 11}},
// 			IsNoCondJump: true,
// 			JumpTo:       lower_ast.CFGReturnTarget,
// 		},
// 		{
// 			Sequence:     lower_ast.SuffixOperatesSequence{lower_ast.LiterialExpression{Value: 22}},
// 			IsNoCondJump: true,
// 			JumpTo:       lower_ast.CFGReturnTarget,
// 		},
// 	}

// 	hopsi, env := compileWithLocalValues(map[string]any{"flag": false}, graph)
// 	hops := hopsi.(HybridOperationsAndAuxSlotSize)
// 	if got, want := describeHybridOperations(hops), "jumpTrue->2, jump->ret, jump->ret"; got != want {
// 		t.Fatalf("compiled operations mismatch\ngot:  %s\nwant: %s", got, want)
// 	}

// 	ret, _ := runCompiledLowerAst(t, hops, env)
// 	if ret != 11 {
// 		t.Fatalf("return value with false flag = %v, want 11", ret)
// 	}
// 	env.LocalAndFreeVars[0].Value = true
// 	ret, _ = runCompiledLowerAst(t, hops, env)
// 	if ret != 22 {
// 		t.Fatalf("return value with true flag = %v, want 22", ret)
// 	}
// }

// func TestCompileLowerAstAppendsJumpWhenFalseDoesNotFallthrough(t *testing.T) {
// 	graph := lower_ast.LowerAst{
// 		{
// 			Sequence: lower_ast.SuffixOperatesSequence{lower_ast.IdentifierExpression{
// 				IdentifierName: "enterLoop",
// 				Scope:          ast.IdentifierScopeLocal,
// 			}},
// 			IsNoCondJump: false,
// 			JumpTo:       1,
// 			Aux:          3,
// 		},
// 		{
// 			Sequence:     lower_ast.SuffixOperatesSequence{lower_ast.LiterialExpression{Value: "body"}},
// 			IsNoCondJump: true,
// 			JumpTo:       2,
// 		},
// 		{
// 			Sequence: lower_ast.SuffixOperatesSequence{lower_ast.IdentifierExpression{
// 				IdentifierName: "keepLoop",
// 				Scope:          ast.IdentifierScopeLocal,
// 			}},
// 			IsNoCondJump: false,
// 			JumpTo:       1,
// 			Aux:          3,
// 		},
// 		{
// 			Sequence:     lower_ast.SuffixOperatesSequence{lower_ast.LiterialExpression{Value: 99}},
// 			IsNoCondJump: true,
// 			JumpTo:       lower_ast.CFGReturnTarget,
// 		},
// 	}

// 	hopsi, env := compileWithLocalValues(map[string]any{
// 		"enterLoop": true,
// 		"keepLoop":  false,
// 	}, graph)
// 	hops := hopsi.(HybridOperationsAndAuxSlotSize)
// 	if got, want := describeHybridOperations(hops), "jumpTrue->4, jump->ret, jumpTrue->4, jump->1, jump->2"; got != want {
// 		t.Fatalf("compiled operations mismatch\ngot:  %s\nwant: %s", got, want)
// 	}

// 	ret, _ := runCompiledLowerAst(t, hops, env)
// 	if ret != 99 {
// 		t.Fatalf("return value = %v, want 99", ret)
// 	}
// }

// func TestCompileNormalOperateSupportsLiteralPrefixListAndMap(t *testing.T) {
// 	ast := ast.ListLiteral{
// 		Elements: []ast.ExpressionNode{
// 			ast.PrefixExpression{
// 				Operator: ast.MINUS,
// 				Right:    ast.Literial{Value: 3},
// 			},
// 			ast.InfixExpression{
// 				Operator: ast.PLUS,
// 				Left:     ast.Literial{Value: 4},
// 				Right:    ast.Literial{Value: 5},
// 			},
// 			ast.MapLiteral{
// 				Pairs: [][2]ast.ExpressionNode{
// 					{ast.Literial{Value: "sum"}, ast.InfixExpression{
// 						Operator: ast.PLUS,
// 						Left:     ast.Literial{Value: 6},
// 						Right:    ast.Literial{Value: 7},
// 					}},
// 					{ast.Literial{Value: "neg"}, ast.PrefixExpression{
// 						Operator: ast.MINUS,
// 						Right:    ast.Literial{Value: 8},
// 					}},
// 				},
// 			},
// 		},
// 	}

// 	hops, env := compileExpressionWithLocalValues(t, nil, ast)
// 	env = runHybridOperations(t, hops, env)

// 	listPtr, ok := env.Vars[0].(*[]any)
// 	if !ok {
// 		t.Fatalf("result type = %T, want *[]any", env.Vars[0])
// 	}
// 	if len(*listPtr) != 3 {
// 		t.Fatalf("list len = %d, want 3", len(*listPtr))
// 	}
// 	if (*listPtr)[0] != -3 {
// 		t.Fatalf("list[0] = %v, want -3", (*listPtr)[0])
// 	}
// 	if (*listPtr)[1] != 9 {
// 		t.Fatalf("list[1] = %v, want 9", (*listPtr)[1])
// 	}
// 	mp, ok := (*listPtr)[2].(object.Map)
// 	if !ok {
// 		t.Fatalf("list[2] type = %T, want object.MAP", (*listPtr)[2])
// 	}
// 	if got := mp["sum"]; got != 13 {
// 		t.Fatalf("map[sum] = %v, want 13", got)
// 	}
// 	if got := mp["neg"]; got != -8 {
// 		t.Fatalf("map[neg] = %v, want -8", got)
// 	}
// }

// func TestCompileNormalOperateSupportsAssignments(t *testing.T) {
// 	listValue := object.NewListFromGo([]any{1, 2, 3})
// 	attrMap := object.Map{}
// 	localValues := map[string]any{
// 		"x":      0,
// 		"arr":    &listValue,
// 		"obj":    attrMap,
// 		"value1": 11,
// 		"value2": 22,
// 		"index":  1,
// 	}

// 	ast := ast.StatementsStatement{
// 		Statements: []ast.StatementNode{
// 			ast.ExpressionStatement{
// 				Expression: ast.AssignmentExpression{
// 					Left: ast.LeftIdentifier{
// 						Identifier: ast.Identifier{Value: "x"},
// 						Scope:      ast.IdentifierScopeLocal,
// 					},
// 					Right: newLocalIdenitifer("value1"),
// 				},
// 			},
// 			ast.ExpressionStatement{
// 				Expression: ast.AssignmentExpression{
// 					Left: ast.LeftSetAttribute{
// 						CanSetAttribute: newLocalIdenitifer("obj"),
// 						Attribute:       "answer",
// 					},
// 					Right: newLocalIdenitifer("value2"),
// 				},
// 			},
// 			ast.ReturnStatement{
// 				ReturnValue: ast.AssignmentExpression{
// 					Left: ast.LeftSetIndex{
// 						CanSetIndex: newLocalIdenitifer("arr"),
// 						Index:       newLocalIdenitifer("index"),
// 					},
// 					Right: ast.InfixExpression{
// 						Operator: ast.PLUS,
// 						Left:     newLocalIdenitifer("x"),
// 						Right:    newLocalIdenitifer("value2"),
// 					},
// 				},
// 			},
// 		},
// 	}

// 	last := convert_ast.ConvertAst(ast).(lower_ast.LowerAst)
// 	if len(last) != 1 {
// 		t.Fatalf("block len = %d, want 1", len(last))
// 	}
// 	hopsi, env := compileWithLocalValues(localValues, last[0].Sequence)
// 	hops, ok := hopsi.(HybridOperationsAndAuxSlotSize)
// 	if !ok {
// 		t.Fatalf("CompileAst() returned %T, want hybridOperationsAndAuxSlotSize", hopsi)
// 	}
// 	env = runHybridOperations(t, hops, env)

// 	xSlot := genIdentifierScopsLookup([]IdentiferNamesWithScope{{
// 		symbolNames: []string{"x", "arr", "obj", "value1", "value2", "index"},
// 		scopeName:   ast.IdentifierScopeLocal,
// 	}})[lower_ast.IdentifierExpression{IdentifierName: "x", Scope: ast.IdentifierScopeLocal}]
// 	if got := env.LocalAndFreeVars[xSlot].Value; got != 11 {
// 		t.Fatalf("x = %v, want 11", got)
// 	}
// 	if got := attrMap["answer"]; got != 22 {
// 		t.Fatalf("obj.answer = %v, want 22", got)
// 	}
// 	if got := listValue.GetItem(1); got != 33 {
// 		t.Fatalf("arr[1] = %v, want 33", got)
// 	}
// 	if got := env.Vars[0]; got != 33 {
// 		t.Fatalf("final result = %v, want 33", got)
// 	}
// }

// func TestCompileLowerAstNestedIfAndReturnWithCalls(t *testing.T) {
// 	statement := ast.IfStatement{
// 		ConditionAndConsequence: []ast.ConditionAndConsequence{
// 			{
// 				Cond: newLocalCall("cond1"),
// 				Do: ast.IfStatement{
// 					ConditionAndConsequence: []ast.ConditionAndConsequence{
// 						{
// 							Cond: newLocalCall("cond2"),
// 							Do: ast.ReturnStatement{
// 								ReturnValue: newLocalCall("retA", newLocalIdenitifer("x")),
// 							},
// 						},
// 					},
// 					Alternative: ast.ReturnStatement{
// 						ReturnValue: newLocalCall("retB", newLocalIdenitifer("y")),
// 					},
// 				},
// 			},
// 		},
// 		Alternative: ast.ReturnStatement{
// 			ReturnValue: newLocalCall("retC", newLocalIdenitifer("z")),
// 		},
// 	}

// 	cases := []struct {
// 		name     string
// 		cond1    bool
// 		cond2    bool
// 		wantRet  any
// 		wantCall []string
// 	}{
// 		{
// 			name:     "then-then",
// 			cond1:    true,
// 			cond2:    true,
// 			wantRet:  101,
// 			wantCall: []string{"cond1()", "cond2()", "retA(1)"},
// 		},
// 		{
// 			name:     "then-else",
// 			cond1:    true,
// 			cond2:    false,
// 			wantRet:  202,
// 			wantCall: []string{"cond1()", "cond2()", "retB(2)"},
// 		},
// 		{
// 			name:     "outer-else",
// 			cond1:    false,
// 			cond2:    false,
// 			wantRet:  303,
// 			wantCall: []string{"cond1()", "retC(3)"},
// 		},
// 	}

// 	for _, tc := range cases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			callLog := make([]string, 0, 4)
// 			locals := map[string]any{
// 				"x": 1, "y": 2, "z": 3,
// 				"cond1": func(args []any) any {
// 					callLog = append(callLog, "cond1()")
// 					return tc.cond1
// 				},
// 				"cond2": func(args []any) any {
// 					callLog = append(callLog, "cond2()")
// 					return tc.cond2
// 				},
// 				"retA": func(args []any) any {
// 					callLog = append(callLog, fmt.Sprintf("retA(%v)", args[0]))
// 					return 100 + args[0].(int)
// 				},
// 				"retB": func(args []any) any {
// 					callLog = append(callLog, fmt.Sprintf("retB(%v)", args[0]))
// 					return 200 + args[0].(int)
// 				},
// 				"retC": func(args []any) any {
// 					callLog = append(callLog, fmt.Sprintf("retC(%v)", args[0]))
// 					return 300 + args[0].(int)
// 				},
// 			}

// 			hops, env := compileStatementWithLocalValues(t, locals, statement)
// 			ret, _ := runCompiledLowerAst(t, hops, env)
// 			if ret != tc.wantRet {
// 				t.Fatalf("return value = %v, want %v", ret, tc.wantRet)
// 			}
// 			if !slices.Equal(callLog, tc.wantCall) {
// 				t.Fatalf("call order mismatch\ngot:  %v\nwant: %v", callLog, tc.wantCall)
// 			}
// 		})
// 	}
// }

// func TestCompileLowerAstLoopWithBreakContinueAndCalls(t *testing.T) {
// 	statement := ast.StatementsStatement{
// 		Statements: []ast.StatementNode{
// 			newLocalAssignStatement("sum", ast.Literial{Value: 0}),
// 			newLocalAssignStatement("i", ast.Literial{Value: 0}),
// 			ast.LoopStatement{
// 				Condition: newLocalCall("loopCond", newLocalIdenitifer("i"), newLocalIdenitifer("limit")),
// 				LoopBody: ast.StatementsStatement{
// 					Statements: []ast.StatementNode{
// 						ast.IfStatement{
// 							ConditionAndConsequence: []ast.ConditionAndConsequence{
// 								{
// 									Cond: newLocalCall("shouldSkip", newLocalIdenitifer("i")),
// 									Do:   ast.ContinueStatement{},
// 								},
// 							},
// 						},
// 						ast.IfStatement{
// 							ConditionAndConsequence: []ast.ConditionAndConsequence{
// 								{
// 									Cond: newLocalCall("shouldStop", newLocalIdenitifer("i")),
// 									Do:   ast.BreakStatement{},
// 								},
// 							},
// 						},
// 						newLocalAssignStatement("sum", ast.InfixExpression{
// 							Operator: ast.PLUS,
// 							Left:     newLocalIdenitifer("sum"),
// 							Right:    newLocalCall("valueOf", newLocalIdenitifer("i")),
// 						}),
// 					},
// 				},
// 				AfterEachLoop: newLocalAssignStatement("i", newLocalCall("next", newLocalIdenitifer("i"))),
// 			},
// 			ast.ReturnStatement{
// 				ReturnValue: newLocalCall("finish", newLocalIdenitifer("sum")),
// 			},
// 		},
// 	}

// 	callLog := make([]string, 0, 32)
// 	locals := map[string]any{
// 		"sum":   0,
// 		"i":     0,
// 		"limit": 5,
// 		"loopCond": func(args []any) any {
// 			i, limit := args[0].(int), args[1].(int)
// 			callLog = append(callLog, fmt.Sprintf("loopCond(%d,%d)", i, limit))
// 			return i < limit
// 		},
// 		"shouldSkip": func(args []any) any {
// 			i := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("shouldSkip(%d)", i))
// 			return i == 1
// 		},
// 		"shouldStop": func(args []any) any {
// 			i := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("shouldStop(%d)", i))
// 			return i == 4
// 		},
// 		"valueOf": func(args []any) any {
// 			i := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("valueOf(%d)", i))
// 			return i * 10
// 		},
// 		"next": func(args []any) any {
// 			i := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("next(%d)", i))
// 			return i + 1
// 		},
// 		"finish": func(args []any) any {
// 			sum := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("finish(%d)", sum))
// 			return sum + 1
// 		},
// 	}

// 	hops, env := compileStatementWithLocalValues(t, locals, statement)
// 	ret, _ := runCompiledLowerAst(t, hops, env)
// 	if ret != 51 {
// 		t.Fatalf("return value = %v, want 51", ret)
// 	}
// 	wantCallLog := []string{
// 		"loopCond(0,5)",
// 		"shouldSkip(0)",
// 		"shouldStop(0)",
// 		"valueOf(0)",
// 		"next(0)",
// 		"loopCond(1,5)",
// 		"shouldSkip(1)",
// 		"next(1)",
// 		"loopCond(2,5)",
// 		"shouldSkip(2)",
// 		"shouldStop(2)",
// 		"valueOf(2)",
// 		"next(2)",
// 		"loopCond(3,5)",
// 		"shouldSkip(3)",
// 		"shouldStop(3)",
// 		"valueOf(3)",
// 		"next(3)",
// 		"loopCond(4,5)",
// 		"shouldSkip(4)",
// 		"shouldStop(4)",
// 		"finish(50)",
// 	}
// 	if !slices.Equal(callLog, wantCallLog) {
// 		t.Fatalf("call order mismatch\ngot:  %v\nwant: %v", callLog, wantCallLog)
// 	}
// }

// func TestCompileLowerAstNestedLoopsWithBreakContinueAndCalls(t *testing.T) {
// 	statement := ast.StatementsStatement{
// 		Statements: []ast.StatementNode{
// 			newLocalAssignStatement("sum", ast.Literial{Value: 0}),
// 			newLocalAssignStatement("i", ast.Literial{Value: 0}),
// 			ast.LoopStatement{
// 				Condition: newLocalCall("outerCond", newLocalIdenitifer("i")),
// 				LoopBody: ast.StatementsStatement{
// 					Statements: []ast.StatementNode{
// 						newLocalAssignStatement("j", ast.Literial{Value: 0}),
// 						ast.LoopStatement{
// 							Condition: newLocalCall("innerCond", newLocalIdenitifer("j")),
// 							LoopBody: ast.StatementsStatement{
// 								Statements: []ast.StatementNode{
// 									ast.IfStatement{
// 										ConditionAndConsequence: []ast.ConditionAndConsequence{
// 											{
// 												Cond: newLocalCall("skipPair", newLocalIdenitifer("i"), newLocalIdenitifer("j")),
// 												Do:   ast.ContinueStatement{},
// 											},
// 										},
// 									},
// 									ast.IfStatement{
// 										ConditionAndConsequence: []ast.ConditionAndConsequence{
// 											{
// 												Cond: newLocalCall("stopInner", newLocalIdenitifer("i"), newLocalIdenitifer("j")),
// 												Do:   ast.BreakStatement{},
// 											},
// 										},
// 									},
// 									newLocalAssignStatement("sum", ast.InfixExpression{
// 										Operator: ast.PLUS,
// 										Left:     newLocalIdenitifer("sum"),
// 										Right: newLocalCall("pairValue",
// 											newLocalIdenitifer("i"),
// 											newLocalIdenitifer("j"),
// 										),
// 									}),
// 								},
// 							},
// 							AfterEachLoop: newLocalAssignStatement("j", newLocalCall("nextInner", newLocalIdenitifer("j"))),
// 						},
// 						ast.IfStatement{
// 							ConditionAndConsequence: []ast.ConditionAndConsequence{
// 								{
// 									Cond: newLocalCall("stopOuter", newLocalIdenitifer("i")),
// 									Do:   ast.BreakStatement{},
// 								},
// 							},
// 						},
// 					},
// 				},
// 				AfterEachLoop: newLocalAssignStatement("i", newLocalCall("nextOuter", newLocalIdenitifer("i"))),
// 			},
// 			ast.ReturnStatement{
// 				ReturnValue: newLocalCall("finish", newLocalIdenitifer("sum")),
// 			},
// 		},
// 	}

// 	callLog := make([]string, 0, 64)
// 	locals := map[string]any{
// 		"sum": 0, "i": 0, "j": 0,
// 		"outerCond": func(args []any) any {
// 			i := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("outerCond(%d)", i))
// 			return i < 3
// 		},
// 		"innerCond": func(args []any) any {
// 			j := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("innerCond(%d)", j))
// 			return j < 3
// 		},
// 		"skipPair": func(args []any) any {
// 			i, j := args[0].(int), args[1].(int)
// 			callLog = append(callLog, fmt.Sprintf("skipPair(%d,%d)", i, j))
// 			return i == 0 && j == 1
// 		},
// 		"stopInner": func(args []any) any {
// 			i, j := args[0].(int), args[1].(int)
// 			callLog = append(callLog, fmt.Sprintf("stopInner(%d,%d)", i, j))
// 			return i == 1 && j == 2
// 		},
// 		"pairValue": func(args []any) any {
// 			i, j := args[0].(int), args[1].(int)
// 			callLog = append(callLog, fmt.Sprintf("pairValue(%d,%d)", i, j))
// 			return i*10 + j
// 		},
// 		"nextInner": func(args []any) any {
// 			j := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("nextInner(%d)", j))
// 			return j + 1
// 		},
// 		"stopOuter": func(args []any) any {
// 			i := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("stopOuter(%d)", i))
// 			return i == 1
// 		},
// 		"nextOuter": func(args []any) any {
// 			i := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("nextOuter(%d)", i))
// 			return i + 1
// 		},
// 		"finish": func(args []any) any {
// 			sum := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("finish(%d)", sum))
// 			return sum
// 		},
// 	}

// 	hops, env := compileStatementWithLocalValues(t, locals, statement)
// 	ret, _ := runCompiledLowerAst(t, hops, env)
// 	if ret != 23 {
// 		t.Fatalf("return value = %v, want 23", ret)
// 	}
// 	wantCallLog := []string{
// 		"outerCond(0)",
// 		"innerCond(0)",
// 		"skipPair(0,0)",
// 		"stopInner(0,0)",
// 		"pairValue(0,0)",
// 		"nextInner(0)",
// 		"innerCond(1)",
// 		"skipPair(0,1)",
// 		"nextInner(1)",
// 		"innerCond(2)",
// 		"skipPair(0,2)",
// 		"stopInner(0,2)",
// 		"pairValue(0,2)",
// 		"nextInner(2)",
// 		"innerCond(3)",
// 		"stopOuter(0)",
// 		"nextOuter(0)",
// 		"outerCond(1)",
// 		"innerCond(0)",
// 		"skipPair(1,0)",
// 		"stopInner(1,0)",
// 		"pairValue(1,0)",
// 		"nextInner(0)",
// 		"innerCond(1)",
// 		"skipPair(1,1)",
// 		"stopInner(1,1)",
// 		"pairValue(1,1)",
// 		"nextInner(1)",
// 		"innerCond(2)",
// 		"skipPair(1,2)",
// 		"stopInner(1,2)",
// 		"stopOuter(1)",
// 		"finish(23)",
// 	}
// 	if !slices.Equal(callLog, wantCallLog) {
// 		t.Fatalf("call order mismatch\ngot:  %v\nwant: %v", callLog, wantCallLog)
// 	}
// }

// func TestCompileAstCallBecomesNormalOpWhenAsyncDisabled(t *testing.T) {
// 	ast := ast.InfixExpression{
// 		Operator: ast.PLUS,
// 		Left:     newLocalIdenitifer("x0"),
// 		Right: newLocalCall("fn",
// 			ast.InfixExpression{
// 				Operator: ast.PLUS,
// 				Left:     newLocalIdenitifer("x1"),
// 				Right:    newLocalIdenitifer("x2"),
// 			},
// 			newLocalIdenitifer("x3"),
// 		),
// 	}

// 	hops, env := compileExpressionWithLocalValuesAllowAsync(t, map[string]any{
// 		"x0": 1, "x1": 2, "x2": 3, "x3": 4,
// 		"fn": func(args []any) any {
// 			if len(args) != 2 {
// 				t.Fatalf("fn expects 2 args, got %d", len(args))
// 			}
// 			return args[0].(int)*10 + args[1].(int)
// 		},
// 	}, ast, false)

// 	assertNoAsyncOps(t, hops)
// 	if got := describeHybridOperations(hops); got != "keep@0" {
// 		t.Fatalf("compiled operations = %s, want keep@0", got)
// 	}

// 	env = runHybridOperations(t, hops, env)
// 	if got := env.Vars[0]; got != 55 {
// 		t.Fatalf("final result = %v, want 55", got)
// 	}
// }

// func TestCompileLowerAstExecutesCallsWhenAsyncDisabled(t *testing.T) {
// 	statement := ast.StatementsStatement{
// 		Statements: []ast.StatementNode{
// 			newLocalAssignStatement("sum", ast.Literial{Value: 0}),
// 			newLocalAssignStatement("i", ast.Literial{Value: 0}),
// 			ast.LoopStatement{
// 				Condition: newLocalCall("loopCond", newLocalIdenitifer("i")),
// 				LoopBody: ast.StatementsStatement{
// 					Statements: []ast.StatementNode{
// 						ast.IfStatement{
// 							ConditionAndConsequence: []ast.ConditionAndConsequence{
// 								{
// 									Cond: newLocalCall("shouldSkip", newLocalIdenitifer("i")),
// 									Do:   ast.ContinueStatement{},
// 								},
// 							},
// 						},
// 						newLocalAssignStatement("sum", newLocalCall("addValue", newLocalIdenitifer("sum"), newLocalIdenitifer("i"))),
// 					},
// 				},
// 				AfterEachLoop: newLocalAssignStatement("i", newLocalCall("next", newLocalIdenitifer("i"))),
// 			},
// 			ast.IfStatement{
// 				ConditionAndConsequence: []ast.ConditionAndConsequence{
// 					{
// 						Cond: newLocalCall("isBig", newLocalIdenitifer("sum")),
// 						Do: ast.ReturnStatement{
// 							ReturnValue: newLocalCall("retTrue", newLocalIdenitifer("sum")),
// 						},
// 					},
// 				},
// 				Alternative: ast.ReturnStatement{
// 					ReturnValue: newLocalCall("retFalse", newLocalIdenitifer("sum")),
// 				},
// 			},
// 		},
// 	}

// 	callLog := make([]string, 0, 32)
// 	hops, env := compileStatementWithLocalValuesAllowAsync(t, map[string]any{
// 		"sum": 0,
// 		"i":   0,
// 		"loopCond": func(args []any) any {
// 			i := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("loopCond(%d)", i))
// 			return i < 4
// 		},
// 		"shouldSkip": func(args []any) any {
// 			i := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("shouldSkip(%d)", i))
// 			return i == 1
// 		},
// 		"addValue": func(args []any) any {
// 			sum, i := args[0].(int), args[1].(int)
// 			callLog = append(callLog, fmt.Sprintf("addValue(%d,%d)", sum, i))
// 			return sum + i*10
// 		},
// 		"next": func(args []any) any {
// 			i := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("next(%d)", i))
// 			return i + 1
// 		},
// 		"isBig": func(args []any) any {
// 			sum := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("isBig(%d)", sum))
// 			return sum > 20
// 		},
// 		"retTrue": func(args []any) any {
// 			sum := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("retTrue(%d)", sum))
// 			return sum + 100
// 		},
// 		"retFalse": func(args []any) any {
// 			sum := args[0].(int)
// 			callLog = append(callLog, fmt.Sprintf("retFalse(%d)", sum))
// 			return sum + 200
// 		},
// 	}, statement, false)

// 	assertNoAsyncOps(t, hops)
// 	ret, _ := runCompiledLowerAst(t, hops, env)
// 	if ret != 150 {
// 		t.Fatalf("return value = %v, want 150", ret)
// 	}
// 	wantCallLog := []string{
// 		"loopCond(0)",
// 		"shouldSkip(0)",
// 		"addValue(0,0)",
// 		"next(0)",
// 		"loopCond(1)",
// 		"shouldSkip(1)",
// 		"next(1)",
// 		"loopCond(2)",
// 		"shouldSkip(2)",
// 		"addValue(0,2)",
// 		"next(2)",
// 		"loopCond(3)",
// 		"shouldSkip(3)",
// 		"addValue(20,3)",
// 		"next(3)",
// 		"loopCond(4)",
// 		"isBig(50)",
// 		"retTrue(50)",
// 	}
// 	if !slices.Equal(callLog, wantCallLog) {
// 		t.Fatalf("call order mismatch\ngot:  %v\nwant: %v", callLog, wantCallLog)
// 	}
// }

// func TestCompileAstPanicsAtRuntimeOnYieldWhenAsyncDisabled(t *testing.T) {
// 	hops, env := compileExpressionWithLocalValuesAllowAsync(t, nil, ast.YieldExpression{
// 		ReasonValue: ast.InfixExpression{
// 			Operator: ast.PLUS,
// 			Left:     ast.Literial{Value: 1},
// 			Right:    ast.Literial{Value: 2},
// 		},
// 	}, false)

// 	assertNoAsyncOps(t, hops)

// 	defer func() {
// 		if r := recover(); r == nil {
// 			t.Fatal("runHybridOperations() did not panic")
// 		}
// 	}()

// 	runHybridOperations(t, hops, env)
// }

// func TestEnclosureGetCallableReturnsCompiledResult(t *testing.T) {
// 	rootFn := ast.CreateEnclosureExpression{
// 		ParameterSymbols: []ast.Identifier{{Value: "x"}, {Value: "y"}},
// 		LocalSymbols:     []ast.Identifier{{Value: "x"}, {Value: "y"}},
// 		Body: ast.ReturnStatement{
// 			ReturnValue: ast.InfixExpression{
// 				Operator: ast.PLUS,
// 				Left:     newLocalIdenitifer("x"),
// 				Right:    newLocalIdenitifer("y"),
// 			},
// 		},
// 	}

// 	root, _, _ := compileRootEnclosure(t, rootFn, nil)
// 	got := root.GetCallable()([]object.Object{1, 2})
// 	if got != 3 {
// 		t.Fatalf("call result = %v, want 3", got)
// 	}
// }

// func TestEnclosureGetCallableSupportsNestedObjectLanguageClosures(t *testing.T) {
// 	inner := ast.CreateEnclosureExpression{
// 		ParameterSymbols: []ast.Identifier{{Value: "c"}},
// 		LocalSymbols:     []ast.Identifier{{Value: "c"}},
// 		FreeSymbols:      []ast.Identifier{{Value: "a"}, {Value: "b"}},
// 		Body: ast.ReturnStatement{
// 			ReturnValue: ast.InfixExpression{
// 				Operator: ast.PLUS,
// 				Left: ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left:     newFreeIdentifier("a"),
// 					Right:    newFreeIdentifier("b"),
// 				},
// 				Right: newLocalIdenitifer("c"),
// 			},
// 		},
// 	}
// 	middle := ast.CreateEnclosureExpression{
// 		ParameterSymbols: []ast.Identifier{{Value: "b"}},
// 		LocalSymbols:     []ast.Identifier{{Value: "b"}},
// 		FreeSymbols:      []ast.Identifier{{Value: "a"}},
// 		Body: ast.ReturnStatement{
// 			ReturnValue: inner,
// 		},
// 	}
// 	rootFn := ast.CreateEnclosureExpression{
// 		ParameterSymbols: []ast.Identifier{{Value: "a"}},
// 		LocalSymbols:     []ast.Identifier{{Value: "a"}},
// 		Body: ast.ReturnStatement{
// 			ReturnValue: middle,
// 		},
// 	}

// 	root, _, _ := compileRootEnclosure(t, rootFn, nil)
// 	level1 := root.GetCallable()([]object.Object{1})
// 	level2 := callObjectCallable(t, level1, 2)
// 	got := callObjectCallable(t, level2, 3)
// 	if got != 6 {
// 		t.Fatalf("nested closure result = %v, want 6", got)
// 	}
// }

// func TestEnclosureGetCallableCapturesMutatedLocal(t *testing.T) {
// 	inner := ast.CreateEnclosureExpression{
// 		ParameterSymbols: []ast.Identifier{{Value: "b"}},
// 		LocalSymbols:     []ast.Identifier{{Value: "b"}},
// 		FreeSymbols:      []ast.Identifier{{Value: "a"}},
// 		Body: ast.ReturnStatement{
// 			ReturnValue: ast.InfixExpression{
// 				Operator: ast.PLUS,
// 				Left:     newFreeIdentifier("a"),
// 				Right:    newLocalIdenitifer("b"),
// 			},
// 		},
// 	}
// 	rootFn := ast.CreateEnclosureExpression{
// 		ParameterSymbols: []ast.Identifier{{Value: "a"}},
// 		LocalSymbols:     []ast.Identifier{{Value: "a"}, {Value: "f"}},
// 		Body: ast.StatementsStatement{
// 			Statements: []ast.StatementNode{
// 				newLocalAssignStatement("f", inner),
// 				newLocalAssignStatement("a", ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left:     newLocalIdenitifer("a"),
// 					Right:    ast.Literial{Value: 10},
// 				}),
// 				ast.ReturnStatement{
// 					ReturnValue: newLocalIdenitifer("f"),
// 				},
// 			},
// 		},
// 	}

// 	root, _, _ := compileRootEnclosure(t, rootFn, nil)
// 	closure := root.GetCallable()([]object.Object{1})
// 	got := callObjectCallable(t, closure, 2)
// 	if got != 13 {
// 		t.Fatalf("captured closure result = %v, want 13", got)
// 	}
// }

// func TestCompileAstFnCacheReusesNestedFunctionCompilationAcrossSyncAndAsyncPasses(t *testing.T) {
// 	innerMost := ast.CreateEnclosureExpression{
// 		ParameterSymbols: []ast.Identifier{{Value: "z"}},
// 		LocalSymbols:     []ast.Identifier{{Value: "z"}},
// 		FreeSymbols:      []ast.Identifier{{Value: "x"}, {Value: "y"}},
// 		Body: ast.ReturnStatement{
// 			ReturnValue: ast.InfixExpression{
// 				Operator: ast.PLUS,
// 				Left: ast.InfixExpression{
// 					Operator: ast.PLUS,
// 					Left:     newFreeIdentifier("x"),
// 					Right:    newFreeIdentifier("y"),
// 				},
// 				Right: newLocalIdenitifer("z"),
// 			},
// 		},
// 	}
// 	middle := ast.CreateEnclosureExpression{
// 		ParameterSymbols: []ast.Identifier{{Value: "y"}},
// 		LocalSymbols:     []ast.Identifier{{Value: "y"}},
// 		FreeSymbols:      []ast.Identifier{{Value: "x"}},
// 		Body: ast.ReturnStatement{
// 			ReturnValue: innerMost,
// 		},
// 	}
// 	rootFn := ast.CreateEnclosureExpression{
// 		ParameterSymbols: []ast.Identifier{{Value: "x"}},
// 		LocalSymbols:     []ast.Identifier{{Value: "x"}},
// 		Body: ast.ReturnStatement{
// 			ReturnValue: middle,
// 		},
// 	}

// 	proto := convert_ast.ConvertFn(rootFn)
// 	innerLookup := buildInnerLookupForProto(proto)
// 	fnCache := map[string]functionProto{}

// 	CompileFnProtoWithCfg(proto, Lookup{}, nil, fnCache)
// 	cacheSizeAfterCompileFn := len(fnCache)
// 	if cacheSizeAfterCompileFn != 2 {
// 		t.Fatalf("shared tree cache size after CompileFn = %d, want 2", cacheSizeAfterCompileFn)
// 	}

// 	for key, fp := range fnCache {
// 		if len(fp.OperationsNonAsync) == 0 {
// 			t.Fatalf("cached function %q has empty non-async operations", key)
// 		}
// 		if len(fp.OperationsAsync) == 0 {
// 			t.Fatalf("cached function %q has empty async operations", key)
// 		}
// 	}

// 	Compile(proto.Graph, innerLookup, nil, false, fnCache)
// 	if got := len(fnCache); got != cacheSizeAfterCompileFn {
// 		t.Fatalf("shared tree cache size after extra non-async compile = %d, want %d", got, cacheSizeAfterCompileFn)
// 	}
// 	Compile(proto.Graph, innerLookup, nil, true, fnCache)
// 	if got := len(fnCache); got != cacheSizeAfterCompileFn {
// 		t.Fatalf("shared tree cache size after extra async compile = %d, want %d", got, cacheSizeAfterCompileFn)
// 	}
// }

func CompileFib35(t *testing.T, nonAsync bool) *engine.Enclosure {
	fibName := ast.Identifier{Value: "fibonacci"}
	xName := ast.Identifier{Value: "x"}
	fibFn := ast.CreateEnclosureExpression{
		NonAsync:         nonAsync,
		ParameterSymbols: []ast.Identifier{xName},
		LocalSymbols:     []ast.Identifier{xName},
		FreeSymbols:      []ast.Identifier{fibName},
		Body: ast.IfStatement{
			ConditionAndConsequence: []ast.ConditionAndConsequence{
				{
					Cond: ast.InfixExpression{
						Operator: ir_operator.EQ,
						Left:     newLocalIdenitifer("x"),
						Right:    ast.Literial{Value: 0},
					},
					Do: ast.ReturnStatement{ReturnValue: ast.Literial{Value: 0}},
				},
			},
			Alternative: ast.IfStatement{
				ConditionAndConsequence: []ast.ConditionAndConsequence{
					{
						Cond: ast.InfixExpression{
							Operator: ir_operator.EQ,
							Left:     newLocalIdenitifer("x"),
							Right:    ast.Literial{Value: 1},
						},
						Do: ast.ReturnStatement{ReturnValue: ast.Literial{Value: 1}},
					},
				},
				Alternative: ast.ReturnStatement{
					ReturnValue: ast.InfixExpression{
						Operator: ir_operator.PLUS,
						Left: ast.CallExpression{
							Function: newFreeIdentifier("fibonacci"),
							Arguments: []ast.ExpressionNode{
								ast.InfixExpression{
									Operator: ir_operator.MINUS,
									Left:     newLocalIdenitifer("x"),
									Right:    ast.Literial{Value: 1},
								},
							},
						},
						Right: ast.CallExpression{
							Function: newFreeIdentifier("fibonacci"),
							Arguments: []ast.ExpressionNode{
								ast.InfixExpression{
									Operator: ir_operator.MINUS,
									Left:     newLocalIdenitifer("x"),
									Right:    ast.Literial{Value: 2},
								},
							},
						},
					},
				},
			},
		},
	}

	rootFn := ast.CreateEnclosureExpression{
		LocalSymbols: []ast.Identifier{fibName},
		Body: ast.StatementsStatement{
			Statements: []ast.StatementNode{
				newLocalAssignStatement("fibonacci", fibFn),
				ast.ReturnStatement{
					ReturnValue: ast.CallExpression{
						Function: newLocalIdenitifer("fibonacci"),
						Arguments: []ast.ExpressionNode{
							ast.Literial{Value: 35},
						},
					},
				},
			},
		},
	}

	root, _, _ := compileRootEnclosure(t, rootFn, nil)
	return root
}

func TestCompiledFib35Timing(t *testing.T) {
	root := CompileFib35(t, false)
	objectFN := root.GetCallable()
	for range 3 {
		start := time.Now()
		got := objectFN(nil)
		elapsed := time.Since(start)
		if got != object.BoxLit(9227465) {
			t.Fatalf("fib(35) = %v, want 9227465", got)
		}
		t.Logf("fib(35)=%v elapsed=%s", got, elapsed)
	}
}

func TestCompiledFib35AsyncTiming(t *testing.T) {
	root := CompileFib35(t, false)
	objectFN := root.GetAsyncCallable()
	for range 3 {
		start := time.Now()
		ret := objectFN(nil, nil)
		got := ret.(async.YieldByFinish[[]object.Box, object.Box]).Result
		elapsed := time.Since(start)
		if got != object.BoxLit(9227465) {
			t.Fatalf("fib(35) = %v, want 9227465", got)
		}
		t.Logf("fib(35)=%v elapsed=%s", got, elapsed)
	}
}

func TestAsyncSleep(t *testing.T) {
	var makeAsleep, captureYield object.NormalHostFn
	yieldValues := make([]object.Box, 0)

	makeAsleep = func(args []object.Box) object.Box {
		if len(args) != 0 {
			t.Fatalf("asleep args count = %d, want 0", len(args))
		}
		sleep := async.YieldBySleep[[]object.Box, object.Box]{AwakeAt: time.Now().Add(time.Millisecond * 10)}
		return object.BoxAny(sleep)
	}
	captureYield = func(args []object.Box) object.Box {
		if len(args) != 1 {
			t.Fatalf("captureYield args count = %d, want 1", len(args))
		}
		if args[0].BasicType != object.BasicTypCustom {
			t.Fatalf("yield result basic type = %d, want %d(custom)", args[0].BasicType, object.BasicTypCustom)
		}
		// resumeAt, ok := object.UnBoxAny(args[0]).(time.Time)
		// if !ok {
		// 	t.Fatalf("yield result type = %T, want time.Time", object.UnBoxAny(args[0]))
		// }
		// if resumeAt.IsZero() {
		// 	t.Fatal("yield result should not be zero time")
		// }
		yieldValues = append(yieldValues, object.Nil)
		return object.Nil
	}

	childFnName := ast.Identifier{Value: "testFn"}
	asleepFnName := ast.Identifier{Value: "asleep"}
	captureYieldFnName := ast.Identifier{Value: "captureYield"}
	firstYieldName := ast.Identifier{Value: "firstYield"}
	secondYieldName := ast.Identifier{Value: "secondYield"}

	globals := map[string]object.Box{
		"asleep":       object.BoxAny(makeAsleep),
		"captureYield": object.BoxAny(captureYield),
	}

	awaitSleepFn := ast.CreateEnclosureExpression{
		NonAsync:     false,
		LocalSymbols: []ast.Identifier{firstYieldName, secondYieldName},
		Body: ast.StatementsStatement{
			Statements: []ast.StatementNode{
				newLocalAssignStatement("firstYield", ast.YieldExpression{
					ReasonValue: ast.CallExpression{
						Function: ast.IdentifierExpression{
							Identifier: asleepFnName,
							Scope:      ast.IdentifierScopeGlobal,
						},
					},
				}),
				ast.ExpressionStatement{
					Expression: ast.CallExpression{
						Function: ast.IdentifierExpression{
							Identifier: captureYieldFnName,
							Scope:      ast.IdentifierScopeGlobal,
						},
						Arguments: []ast.ExpressionNode{
							ast.IdentifierExpression{Identifier: firstYieldName, Scope: ast.IdentifierScopeLocal},
						},
					},
				},
				newLocalAssignStatement("secondYield", ast.YieldExpression{
					ReasonValue: ast.CallExpression{
						Function: ast.IdentifierExpression{
							Identifier: asleepFnName,
							Scope:      ast.IdentifierScopeGlobal,
						},
					},
				}),
				ast.ExpressionStatement{
					Expression: ast.CallExpression{
						Function: ast.IdentifierExpression{
							Identifier: captureYieldFnName,
							Scope:      ast.IdentifierScopeGlobal,
						},
						Arguments: []ast.ExpressionNode{
							ast.IdentifierExpression{Identifier: secondYieldName, Scope: ast.IdentifierScopeLocal},
						},
					},
				},
			},
		},
	}

	rootFn := ast.CreateEnclosureExpression{
		LocalSymbols: []ast.Identifier{childFnName},
		Body: ast.StatementsStatement{
			Statements: []ast.StatementNode{
				newLocalAssignStatement("testFn", awaitSleepFn),
				ast.ReturnStatement{
					ReturnValue: ast.CallExpression{
						Function:  newLocalIdenitifer("testFn"),
						Arguments: []ast.ExpressionNode{},
					},
				},
			},
		},
	}
	enc, _, _ := compileRootEnclosure(t, rootFn, globals)
	asyncFn := enc.GetAsyncCallable()

	runner := async.NewEventLoopRunner[[]object.Box, object.Box]()
	runner.EventLoop.CreateAndAddTask(true, asyncFn, nil, nil, object.AsyncShim)
	runner.RunUntilComplete()

	if len(yieldValues) != 2 {
		t.Fatalf("captured yield value count = %d, want 2", len(yieldValues))
	}
}

func TestAwaitAsyncSleep(t *testing.T) {
	type asyncPayload struct {
		Count int
		Label string
	}

	var makePayload, captureResult object.NormalHostFn
	var asleep object.AsyncHostFn
	var capturedArgPayload asyncPayload
	var capturedResult asyncPayload
	asyncCallCount := 0
	captureResultCount := 0

	makePayload = func(args []object.Box) object.Box {
		if len(args) != 0 {
			t.Fatalf("makePayload args count = %d, want 0", len(args))
		}
		return object.BoxAny(asyncPayload{Count: 3, Label: "seed"})
	}
	captureResult = func(args []object.Box) object.Box {
		if len(args) != 1 {
			t.Fatalf("captureResult args count = %d, want 1", len(args))
		}
		if args[0].BasicType != object.BasicTypCustom {
			t.Fatalf("async return basic type = %d, want %d(custom)", args[0].BasicType, object.BasicTypCustom)
		}
		result, ok := object.UnBoxAny(args[0]).(asyncPayload)
		if !ok {
			t.Fatalf("async return type = %T, want asyncPayload", object.UnBoxAny(args[0]))
		}
		capturedResult = result
		captureResultCount++
		return object.Nil
	}
	asleep = func(handle object.AsyncHandleType, args []object.Box) object.AsyncYieldReason {
		asyncCallCount++
		if len(args) != 3 {
			t.Fatalf("asleep args count = %d, want 3", len(args))
		}
		firstArg := object.UnBoxInt(args[0])
		if firstArg != 7 {
			t.Fatalf("asleep arg[0] = %d, want 7", firstArg)
		}
		secondArg := object.UnBoxString(args[1])
		if secondArg != "neo" {
			t.Fatalf("asleep arg[1] = %q, want %q", secondArg, "neo")
		}
		payload, ok := object.UnBoxAny(args[2]).(asyncPayload)
		if !ok {
			t.Fatalf("asleep arg[2] type = %T, want asyncPayload", object.UnBoxAny(args[2]))
		}
		capturedArgPayload = payload

		sleep := async.YieldBySleep[[]object.Box, object.Box]{AwakeAt: time.Now().Add(10 * time.Millisecond)}
		return sleep.SuspendAndYield(
			handle.Resume,
			func(handle object.AsyncHandleType, args []object.Box) object.AsyncYieldReason {
				return object.AsyncYieldFinish(object.BoxAny(asyncPayload{
					Count: firstArg + payload.Count,
					Label: secondArg + ":" + payload.Label,
				}))
			},
		)
	}

	childFnName := ast.Identifier{Value: "testFn"}
	asleepFnName := ast.Identifier{Value: "asleep"}
	makePayloadFnName := ast.Identifier{Value: "makePayload"}
	captureResultFnName := ast.Identifier{Value: "captureResult"}

	globals := map[string]object.Box{
		"asleep":        object.BoxAny(asleep),
		"makePayload":   object.BoxAny(makePayload),
		"captureResult": object.BoxAny(captureResult),
	}
	xName := ast.Identifier{Value: "x"}
	awaitSleepFn := ast.CreateEnclosureExpression{
		NonAsync:     false,
		LocalSymbols: []ast.Identifier{xName},
		Body: ast.StatementsStatement{
			Statements: []ast.StatementNode{
				newLocalAssignStatement("x", ast.CallExpression{
					Function: ast.IdentifierExpression{
						Identifier: asleepFnName,
						Scope:      ast.IdentifierScopeGlobal,
					},
					Arguments: []ast.ExpressionNode{
						ast.Literial{Value: 7},
						ast.Literial{Value: "neo"},
						ast.CallExpression{
							Function: ast.IdentifierExpression{
								Identifier: makePayloadFnName,
								Scope:      ast.IdentifierScopeGlobal,
							},
						},
					},
				}),
				ast.ExpressionStatement{
					Expression: ast.CallExpression{
						Function: ast.IdentifierExpression{
							Identifier: captureResultFnName,
							Scope:      ast.IdentifierScopeGlobal,
						},
						Arguments: []ast.ExpressionNode{
							ast.IdentifierExpression{Identifier: xName, Scope: ast.IdentifierScopeLocal},
						},
					},
				},
			},
		},
	}

	rootFn := ast.CreateEnclosureExpression{
		LocalSymbols: []ast.Identifier{childFnName},
		Body: ast.StatementsStatement{
			Statements: []ast.StatementNode{
				newLocalAssignStatement("testFn", awaitSleepFn),
				ast.ReturnStatement{
					ReturnValue: ast.CallExpression{
						Function:  newLocalIdenitifer("testFn"),
						Arguments: []ast.ExpressionNode{},
					},
				},
			},
		},
	}
	enc, _, _ := compileRootEnclosure(t, rootFn, globals)
	asyncFn := enc.GetAsyncCallable()

	runner := async.NewEventLoopRunner[[]object.Box, object.Box]()
	runner.EventLoop.CreateAndAddTask(true, asyncFn, nil, nil, object.AsyncShim)
	runner.RunUntilComplete()

	if asyncCallCount != 1 {
		t.Fatalf("async host function call count = %d, want 1", asyncCallCount)
	}
	if captureResultCount != 1 {
		t.Fatalf("captureResult call count = %d, want 1", captureResultCount)
	}
	if capturedArgPayload != (asyncPayload{Count: 3, Label: "seed"}) {
		t.Fatalf("async arg payload = %#v, want %#v", capturedArgPayload, asyncPayload{Count: 3, Label: "seed"})
	}
	if capturedResult != (asyncPayload{Count: 10, Label: "neo:seed"}) {
		t.Fatalf("async result payload = %#v, want %#v", capturedResult, asyncPayload{Count: 10, Label: "neo:seed"})
	}
}
