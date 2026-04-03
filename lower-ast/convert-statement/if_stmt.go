package convert_statement

import ast "xex/ast"

func (builder *cfgBuilder) lowerIfStatement(statement ast.IfStatement, current int32) (nextBlockIndx int32, isTerminated bool) {
	if len(statement.ConditionAndConsequence) == 0 {
		panic("empty condition and consequence")
	}

	// 当代码仍然有可能执行到 if 之外时，couldFallthrough = true
	// hasFallthrough 为 false 代表所有分支都已经终结，例如每个分支都是 return。
	var fallThroughblock int32
	couldFallthrough := false

	// condBlock 总是指向“当前还没 lower 的条件判断块”。
	// 对于 else-if，它会不断推进到上一个条件的 false 分支。
	condBlock := current

	for i, cc := range statement.ConditionAndConsequence {
		// 注入当前分支的条件判断语句
		builder.graph[condBlock].Sequence = append(builder.graph[condBlock].Sequence, cc.Cond)

		trueTarget := builder.newBlock()
		var falseTarget int32

		if i == len(statement.ConditionAndConsequence)-1 && isEmptyStatement(statement.Alternative) {
			// 当最后一个且 没有 else 语句时，false 时直接跳出 if
			fallThroughblock = builder.newBlock()
			couldFallthrough = true
			falseTarget = fallThroughblock
		} else {
			falseTarget = builder.newBlock()
		}

		builder.terminateCond(condBlock, trueTarget, falseTarget)
		bodyExit, bodyTerminated := builder.lowerStatement(cc.Do, trueTarget)

		if !bodyTerminated {
			// 如果 body 可能跳出 if 那么必须保证 后续 block 存在
			if !couldFallthrough {
				fallThroughblock = builder.newBlock()
				couldFallthrough = true
			}
			// 指向后续 block
			builder.terminateAlways(bodyExit, fallThroughblock)
		}

		condBlock = falseTarget
	}

	if isEmptyStatement(statement.Alternative) {
		// 如果 else 不存在
		return fallThroughblock, !couldFallthrough
	} else {
		// 如果 else 存在
		altExit, altTerminated := builder.lowerStatement(statement.Alternative, condBlock)
		// 如果 alt 没有后续
		if altTerminated {
			return fallThroughblock, !couldFallthrough
		}
		if !couldFallthrough {
			fallThroughblock = builder.newBlock()
			couldFallthrough = true
		}
		builder.terminateAlways(altExit, fallThroughblock)
		return fallThroughblock, false
	}
}
