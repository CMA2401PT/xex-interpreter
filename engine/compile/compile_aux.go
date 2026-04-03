package compile

import (
	"slices"
	ast "xex/ast"
	"xex/engine"
	lower_ast "xex/lower-ast"
	"xex/object"
)

var litTrue = func(e engine.Env) object.Box {
	return object.True
}
var litFalse = func(e engine.Env) object.Box {
	return object.False
}
var litNil = func(e engine.Env) object.Box {
	return object.Empty
}

var ab0 = func(e engine.Env) object.Box { return e.AuxSlots[0] }
var ab1 = func(e engine.Env) object.Box { return e.AuxSlots[1] }
var ab2 = func(e engine.Env) object.Box { return e.AuxSlots[2] }
var ab3 = func(e engine.Env) object.Box { return e.AuxSlots[3] }
var ab4 = func(e engine.Env) object.Box { return e.AuxSlots[4] }
var ab5 = func(e engine.Env) object.Box { return e.AuxSlots[5] }
var ab6 = func(e engine.Env) object.Box { return e.AuxSlots[6] }
var ab7 = func(e engine.Env) object.Box { return e.AuxSlots[7] }
var ab8 = func(e engine.Env) object.Box { return e.AuxSlots[8] }

func newAuxSlotBridge(slot int) compileValue {
	slotCopy := slot
	var op engine.NormalSuffixOperate
	switch slot {
	default:
		op = func(e engine.Env) object.Box { return e.AuxSlots[slotCopy] }
	case 0:
		op = ab0
	case 1:
		op = ab1
	case 2:
		op = ab2
	case 3:
		op = ab3
	case 4:
		op = ab4
	case 5:
		op = ab5
	case 6:
		op = ab6
	case 7:
		op = ab7
	case 8:
		op = ab8
	}
	return compileValue{
		op:         op,
		isFallback: true,
	}
}

func lowerSpecialOperands(
	valueStack *compileValueStack,
	operations *[]engine.HybirdOperation,
	auxSlotMax *int,
	operandCount int,
) (operandBase int) {
	// 倒出现在的操作数
	ops, _ := valueStack.TakeAndCopyRest()
	// 把操作数分成两半，
	// 一半是不需要的但是必须被fallback的，
	// 一半是需要的，和特殊操作相关的
	fallbackOps := ops[:len(ops)-operandCount]
	specialOps := ops[len(ops)-operandCount:]
	// 先对不需要的操作数进行退化，保证其执行顺序
	for slotI, op := range fallbackOps {
		if op.isFallback {
			// 已经退化了，不需要进行操作
			valueStack.Put(op)
			continue
		}
		// 退化这个操作
		*operations = append(*operations, engine.NewHybirdOperationGraphKeep(op.op, slotI))
		*auxSlotMax = max(*auxSlotMax, slotI+1)
		// 后续操作需要用这个值的时候，需要使用一个桥接函数
		valueStack.Put(newAuxSlotBridge(slotI))
	}
	// 对需要的操作数，全部退化到生成值到辅助栈上
	operandBase = len(fallbackOps)
	for offset, op := range specialOps {
		if op.isFallback {
			// 已经退化了，不需要进行操作
			continue
		}
		slotI := operandBase + offset
		*auxSlotMax = max(*auxSlotMax, slotI+1)
		// 退化这个操作
		*operations = append(*operations, engine.NewHybirdOperationGraphKeep(op.op, slotI))
	}
	return operandBase
}

type IdentiferNamesWithScope struct {
	symbolNames []string
	scopeName   ast.IdentifierScope
	noSort      bool
}

func genIdentifierScopsLookup(scopes []IdentiferNamesWithScope) (lookup Lookup) {
	lookup = Lookup{}
	i := 0
	for _, symbolAndScopeName := range scopes {
		names := slices.Clone(symbolAndScopeName.symbolNames)
		if !symbolAndScopeName.noSort {
			slices.Sort(names)
		}
		for _, name := range names {
			lookup[lower_ast.IdentifierExpression{
				IdentifierName: name,
				Scope:          symbolAndScopeName.scopeName,
			}] = i
			i += 1
		}
	}
	return lookup
}
