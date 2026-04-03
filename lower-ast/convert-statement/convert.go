package convert_statement

import (
	"fmt"
	"hash/fnv"
	ast "xex/ast"
	lower_ast "xex/lower-ast"
)

type Expressions []ast.ExpressionNode

func (es Expressions) IdentString(level int) string {
	out := ""
	for _, en := range es {
		out += fmt.Sprintf("%v%v\n", ast.IdentToString(level), en.IdentString(level))
	}
	return out
}

func (es Expressions) Hash() uint64 {
	hasher := fnv.New64a()
	for _, expression := range es {
		if expression == nil {
			_, _ = hasher.Write([]byte{0})
			continue
		}
		_, _ = hasher.Write([]byte(expression.String()))
		_, _ = hasher.Write([]byte{'\n'})
	}
	return hasher.Sum64()
}

type Graph = lower_ast.CFGraph[Expressions]
type Block = lower_ast.CFGBlock[Expressions]

func StatementToCFGraph(statement ast.StatementNode) Graph {
	builder := cfgBuilder{}
	entry := builder.newBlock()
	currentBlock, terminated := builder.lowerStatement(statement, entry)
	if !terminated {
		builder.graph[currentBlock].Sequence = append(builder.graph[currentBlock].Sequence, ast.Literial{Value: nil})
		builder.terminateAlways(currentBlock, lower_ast.CFGReturnTarget)
	}
	return builder.graph
}

type cfgLoopScope struct {
	breakTarget    int32
	continueTarget int32
}

type cfgBuilder struct {
	graph      Graph
	loopScopes []cfgLoopScope
}

func (builder *cfgBuilder) newBlock() int32 {
	builder.graph = append(builder.graph, Block{})
	return int32(len(builder.graph) - 1)
}

func (builder *cfgBuilder) terminateAlways(blockID, jumpTo int32) {
	builder.graph[blockID].IsNoCondJump = true
	builder.graph[blockID].JumpTo = jumpTo
	builder.graph[blockID].Aux = 0
}

func (builder *cfgBuilder) terminateCond(blockID, jumpTo, aux int32) {
	builder.graph[blockID].IsNoCondJump = false
	builder.graph[blockID].JumpTo = jumpTo
	builder.graph[blockID].Aux = aux
}

// current 是当前 statement 应该继续向哪个 block 追加内容。
// 返回值含义:
// 1. nextBlockIndx: 当前 statement 执行后，后续 statement 应该继续写入哪个 block
// 2. isTerminated: 当前控制流是否已经被终结；例如 return、break、continue 都会终结当前路径
func (builder *cfgBuilder) lowerStatement(statement ast.StatementNode, current int32) (nextBlockIndx int32, isTerminated bool) {
	if isEmptyStatement(statement) {
		// 后续 statement 继续向当前 block 写入
		return current, false
	}
	switch st := statement.(type) {
	case ast.StatementsStatement:
		for _, statement := range st.Statements {
			current, isTerminated = builder.lowerStatement(statement, current)
			if isTerminated {
				return current, true
			}
		}
		return current, isTerminated
	case ast.ExpressionStatement:
		builder.graph[current].Sequence = append(builder.graph[current].Sequence, st.Expression)
		return current, false
	case ast.ReturnStatement:
		returnValue := st.ReturnValue
		if returnValue == nil {
			returnValue = ast.Literial{Value: nil}
		}
		builder.graph[current].Sequence = append(builder.graph[current].Sequence, returnValue)
		builder.terminateAlways(current, lower_ast.CFGReturnTarget)
		return current, true
	case ast.BreakStatement:
		scope := builder.currentLoopScope()
		builder.terminateAlways(current, scope.breakTarget)
		return current, true
	case ast.ContinueStatement:
		scope := builder.currentLoopScope()
		builder.terminateAlways(current, scope.continueTarget)
		return current, true
	case ast.IfStatement:
		return builder.lowerIfStatement(st, current)
	case ast.LoopStatement:
		return builder.lowerLoopStatement(st, current)
	default:
		panic(fmt.Errorf("unsupported statement node: %T", statement))
	}
}

func isEmptyStatement(statement ast.StatementNode) bool {
	if statement == nil {
		return true
	}
	_, ok := statement.(ast.EmptyStatement)
	return ok
}

func isLiteralTrue(expression ast.ExpressionNode) bool {
	literal, ok := expression.(ast.Literial)
	return ok && literal.Value == true
}
