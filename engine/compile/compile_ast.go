package compile

import (
	"fmt"
	"slices"
	"xex/engine"
	lower_ast "xex/lower-ast"
	re_arrange "xex/lower-ast/re-arrange"
	"xex/object"
)

type HybridOperationsAndAuxSlotSize struct {
	operations            []engine.HybirdOperation
	auxSlotSizeExceptLast int
	auxSlotSize           int
	lastSpec              specialCase
}

// CompileSeq 调用 compileNormalOperate 将一个序列的 suffix operate 转为
// []HybirdOperation，同时对于一些op进行特殊处理（主要是处理异步操作)
func CompileSeq(seq lower_ast.SuffixOperatesSequence,
	localAndFreeLookup Lookup,
	globalAndImportLookup Lookup,
	allowAsync bool,
	fnCache map[string]FunctionProto,
) HybridOperationsAndAuxSlotSize {
	operations := make([]engine.HybirdOperation, 0)
	valueStack := &compileValueStack{}
	auxSlotMax := 0
	for _, op := range seq {
		switch opt := op.(type) {
		default:
			// 对于通常情况, 正常编译为 NormalSuffix Operate 模型即可
			sop, sc := compileNormalOperate(op, valueStack, localAndFreeLookup, globalAndImportLookup, fnCache)
			valueStack.PutWithSpecial(compileValue{op: sop, isFallback: false}, sc)
		case nil, lower_ast.DropOperate:
			// 丢弃前一个的值，但是我们仍需要转换它，
			// 因为虽然其值被丢弃，但是其副作用仍可能有必要
			last := valueStack.TakeN(1)[0]
			operations = append(operations, engine.NewHybirdOperationGraphDrop(last.op))
		case lower_ast.CallExpression:
			if allowAsync {
				// call 是一种特殊情况，因为可能会遇到需要挂起的协程函数，但是也可能只是普通函数
				// 对于这种情况，需要将图退化到基于栈的操作，
				// 为了保证求值顺序的正确性，即使前面的操作数我们用不到也必须退化他们以保证他们优先被求值
				fnSlot := lowerSpecialOperands(valueStack, &operations, &auxSlotMax, int(opt.ConsumeOperand()))
				// 生成一个调用操作
				operations = append(operations, engine.NewHybirdOperationCall(fnSlot, int(opt.ConsumeOperand())-1))
				// 由于后续操作可能还需要这个函数的结果，所以还需要一个桥接函数从辅助栈中取值
				valueStack.Put(newAuxSlotBridge(fnSlot))
			} else {
				// 如果不允许异步的话，call 算子实际上会变得非常简单，
				// 其被简化为通常调用并可被合并到图中
				operands := valueStack.TakeN(int(opt.ConsumeOperand()))
				fnOp := operands[0].op
				ops := []engine.NormalSuffixOperate{}
				for _, op := range operands[1:] {
					ops = append(ops, op.op)
				}
				callOp := func(e engine.Env) object.Box {
					fn := fnOp(e)
					if fn.BasicType == object.BasicTypEnclosure {
						return e.NonAsyncEvaluator.Eval(object.UnBoxObjEnclosure[*engine.Enclosure](fn), ops, e)
					}
					args := make([]object.Box, len(ops))
					for i, op := range ops {
						args[i] = op(e)
					}
					return object.GetCallable(fn)(args)
				}
				valueStack.Put(compileValue{op: callOp, isFallback: false})
			}

		case lower_ast.YieldExpression:
			if allowAsync {
				// yield 和 call 一样会打断普通图的连续求值：
				// 先将此前必须完成求值的内容 fallback 到辅助槽，
				// 再把 yield reason 放到槽位里，交给执行器处理恢复值。
				yieldSlot := lowerSpecialOperands(valueStack, &operations, &auxSlotMax, int(opt.ConsumeOperand()))
				operations = append(operations, engine.NewHybirdOperationYield(yieldSlot))
				// 恢复后 yield 表达式本身会在同一槽位产生值，因此后续普通操作继续桥接这个槽位即可
				valueStack.Put(newAuxSlotBridge(yieldSlot))
			} else {
				op := valueStack.TakeN(1)[0].op
				valueStack.Put(compileValue{op: func(e engine.Env) object.Box {
					panic(fmt.Errorf("yield is not allow in non-async mode, yield reason: %v", op(e)))
				}, isFallback: false})
			}
		}
	}
	ops, scs := valueStack.TakeAndCopyRest()
	// 因为对 SuffixOperatesSequence 的约束，最后一定收束为一个操作
	if len(ops) != 1 {
		panic(ops)
	}
	op, sc := ops[0], scs[0]
	auxSlotSizeExceptLast := auxSlotMax
	// 对 block / CFG 而言，最后一个值必须显式落在 slot0。
	// 即使当前值已经由 fallback/bridge 形式存在，也仍需要一次 keep@0，
	// 这样上层在把最后一个 GraphKeep 改写为 Jump 时才始终成立。
	auxSlotMax = max(auxSlotMax, 1)
	operations = append(operations, engine.NewHybirdOperationGraphKeep(op.op, 0))
	return HybridOperationsAndAuxSlotSize{operations: operations, auxSlotSize: auxSlotMax, auxSlotSizeExceptLast: auxSlotSizeExceptLast, lastSpec: sc}
}

// CompileCFGBlock 调用 compileSeq 处理一个 block 中的 sequence 部分
// 同时根据 block 的跳转附加跳转信息
func CompileCFGBlock(
	block lower_ast.CFGBlock[lower_ast.SuffixOperatesSequence],
	localAndFreeLookup Lookup,
	globalAndImportLookup Lookup,
	allowAsync bool,
	fnCache map[string]FunctionProto,
) HybridOperationsAndAuxSlotSize {
	if len(block.Sequence) == 0 {
		if !block.IsNoCondJump {
			panic("conditional block must have a condition expression")
		}
		return HybridOperationsAndAuxSlotSize{
			operations:            []engine.HybirdOperation{engine.NewHybirdOperationJumpNoCond(block.JumpTo, litNil)},
			auxSlotSize:           0,
			auxSlotSizeExceptLast: 0,
		}
	}
	seq := CompileSeq(block.Sequence, localAndFreeLookup, globalAndImportLookup, allowAsync, fnCache)
	ops := seq.operations
	lastOp := ops[len(ops)-1]
	if lastOp.OperationType != engine.OpTypeGraphAndKeep || lastOp.ResultSlot != 0 {
		panic(fmt.Errorf("expect last op to be kept in slot0, but get last op: %v", lastOp))
	}
	if block.IsNoCondJump {
		if seq.lastSpec.IsLit && block.JumpTo == -1 {
			ops[len(ops)-1] = engine.NewHybirdOperationJumpNoCondWithLit(block.JumpTo, seq.lastSpec.Lit)
		} else {
			ops[len(ops)-1] = engine.NewHybirdOperationJumpNoCond(block.JumpTo, lastOp.Graph)
		}

		return HybridOperationsAndAuxSlotSize{
			operations:            ops,
			auxSlotSizeExceptLast: seq.auxSlotSizeExceptLast,
			auxSlotSize:           seq.auxSlotSizeExceptLast,
		}
	}
	lastOp.OperationType = engine.OpTypeJumpTrue
	lastOp.JumpTo = block.JumpTo
	ops[len(ops)-1] = lastOp
	return HybridOperationsAndAuxSlotSize{
		operations:            append(ops, engine.NewHybirdOperationJumpNoCond(block.Aux, litNil)),
		auxSlotSizeExceptLast: seq.auxSlotSizeExceptLast,
		auxSlotSize:           seq.auxSlotSizeExceptLast,
	}
}

type fallthroughAnalysisBlock struct {
	Operations    []engine.HybirdOperation
	AuxSlotSize   int
	IsNoCondJump  bool
	TrueTarget    int32
	FalseTarget   int32
	FallthroughOK bool
}

func newFallthroughAnalysisBlock(compiledOps HybridOperationsAndAuxSlotSize) fallthroughAnalysisBlock {
	// 根据最后两个操作进行分析
	compiled := fallthroughAnalysisBlock{
		Operations:    compiledOps.operations,
		AuxSlotSize:   compiledOps.auxSlotSize,
		IsNoCondJump:  compiledOps.operations[len(compiledOps.operations)-1].OperationType == engine.OpTypeJumpNoCond,
		TrueTarget:    compiledOps.operations[len(compiledOps.operations)-1].JumpTo,
		FallthroughOK: false,
	}
	// 当至少有两个操作时，如果最后一个是无条件跳转，倒数第二个是有条件跳转
	// 那么则丢弃最后一个操作并进行 fallthrough 分析
	if len(compiled.Operations) >= 2 &&
		compiledOps.operations[len(compiledOps.operations)-1].OperationType == engine.OpTypeJumpNoCond &&
		compiledOps.operations[len(compiledOps.operations)-2].OperationType == engine.OpTypeJumpTrue {
		compiled.IsNoCondJump = false
		compiled.TrueTarget = compiledOps.operations[len(compiledOps.operations)-2].JumpTo
		compiled.FalseTarget = compiledOps.operations[len(compiledOps.operations)-1].JumpTo
		compiled.Operations = compiled.Operations[:len(compiledOps.operations)-1]
	}
	return compiled
}

func (a fallthroughAnalysisBlock) needAttachJumpNoCond() bool {
	return !a.IsNoCondJump && !a.FallthroughOK
}

// CompileLowerAst 调用 compileCFGBlock 处理多个 block
// 并修改跳转信息将其连接起来
func CompileLowerAst(ast lower_ast.LowerAst,
	localAndFreeLookup Lookup,
	globalAndImportLookup Lookup,
	allowAsync bool,
	fnCache map[string]FunctionProto,
) HybridOperationsAndAuxSlotSize {
	blocks := re_arrange.ReArrangeBlock(ast)
	compiles := make(map[int32]fallthroughAnalysisBlock, len(blocks))
	auxSlotSize := 0
	blockToNext := make(map[int32]int32, len(blocks))
	seqStart := make(map[int32]int32, len(blocks))
	currentSeqLen := 0
	for seqI, block := range blocks {
		compiled := newFallthroughAnalysisBlock(CompileCFGBlock(block.Block, localAndFreeLookup, globalAndImportLookup, allowAsync, fnCache))
		if !compiled.IsNoCondJump && seqI+1 < len(blocks) && blocks[seqI+1].BlockID == compiled.FalseTarget {
			compiled.FallthroughOK = true
		}
		compiles[block.BlockID] = compiled
		auxSlotSize = max(auxSlotSize, compiled.AuxSlotSize)
		seqStart[block.BlockID] = int32(currentSeqLen)
		currentSeqLen += len(compiled.Operations)
		if compiled.needAttachJumpNoCond() {
			currentSeqLen += 1
		}
		if seqI+1 < len(blocks) {
			blockToNext[block.BlockID] = blocks[seqI+1].BlockID
		}
	}
	outSeq := []engine.HybirdOperation{}
	for _, block := range blocks {
		compiled := compiles[block.BlockID]
		seqs := slices.Clone(compiled.Operations)
		lastIdx := len(seqs) - 1
		if seqs[lastIdx].JumpTo != lower_ast.CFGReturnTarget {
			seqs[lastIdx].JumpTo = seqStart[seqs[lastIdx].JumpTo]
		}
		outSeq = append(outSeq, seqs...)
		if !compiled.needAttachJumpNoCond() {
			continue
		}
		outSeq = append(outSeq, engine.HybirdOperation{
			OperationType: engine.OpTypeJumpNoCond,
			JumpTo:        seqStart[compiled.FalseTarget],
		})
	}
	return HybridOperationsAndAuxSlotSize{operations: outSeq, auxSlotSize: auxSlotSize, auxSlotSizeExceptLast: auxSlotSize}
}
