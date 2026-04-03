package convert_statement

import (
	"fmt"
	ast "xex/ast"
)

func (builder *cfgBuilder) lowerLoopStatement(statement ast.LoopStatement, current int32) (nextBlockIndx int32, isTerminated bool) {
	bodyBlock := builder.newBlock()
	// 从第二次开始进入 latch block，latch block 复合了 cond 和 after statement
	needLatchBlock := !isEmptyStatement(statement.AfterEachLoop) || !isLiteralTrue(statement.Condition)
	latchBlock := bodyBlock
	if needLatchBlock {
		latchBlock = builder.newBlock()
	}
	exitBlock := builder.newBlock()

	// 首次进入循环时，condition 可以直接并入 current；
	// 但回边不能跳回 current，否则会重复执行 current 中循环之前的语句。
	if isLiteralTrue(statement.Condition) {
		builder.terminateAlways(current, bodyBlock)
	} else {
		builder.graph[current].Sequence = append(builder.graph[current].Sequence, statement.Condition)
		builder.terminateCond(current, bodyBlock, exitBlock)
	}

	// 将循环的关键信息挂在 loopScopes 上，方便 break, continue 去找
	builder.loopScopes = append(builder.loopScopes, cfgLoopScope{
		breakTarget:    exitBlock,
		continueTarget: latchBlock,
	})
	bodyExit, bodyTerminated := builder.lowerStatement(statement.LoopBody, bodyBlock)
	builder.loopScopes = builder.loopScopes[:len(builder.loopScopes)-1]

	// body 正常结束时，才需要进入回边块。
	if !bodyTerminated {
		builder.terminateAlways(bodyExit, latchBlock)
	}

	if !needLatchBlock {
		// cond 为 true 且 afterEach 为空时，不需要单独的回边块。
		return exitBlock, false
	}

	if !isEmptyStatement(statement.AfterEachLoop) {
		afterEachExit, afterEachTerminated := builder.lowerStatement(statement.AfterEachLoop, latchBlock)
		if afterEachTerminated {
			panic("AfterEachLoop must not terminate control flow")
		}
		if afterEachExit != latchBlock {
			panic("AfterEachLoop must stay in the same block as loop condition")
		}
	}

	// AfterEachLoop 之后立刻执行下一轮条件判断。
	if isLiteralTrue(statement.Condition) {
		builder.terminateAlways(latchBlock, bodyBlock)
	} else {
		builder.graph[latchBlock].Sequence = append(builder.graph[latchBlock].Sequence, statement.Condition)
		builder.terminateCond(latchBlock, bodyBlock, exitBlock)
	}

	// 不管循环体内部是否有 return，循环条件为 false 时都仍然可能走到 exitBlock。
	return exitBlock, false
}

func (builder *cfgBuilder) currentLoopScope() cfgLoopScope {
	if len(builder.loopScopes) == 0 {
		panic(fmt.Errorf("break/continue used outside of loop"))
	}
	return builder.loopScopes[len(builder.loopScopes)-1]
}
