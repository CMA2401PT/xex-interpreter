package compile

import (
	"fmt"
	"strconv"
	ast "xex/ast"
	"xex/engine"
	lower_ast "xex/lower-ast"
	"xex/object"
)

func compileNormalOperate(
	op lower_ast.SuffixOperate,
	valueStack *compileValueStack,
	localAndFreeLookup Lookup,
	globalAndImportLookup Lookup,
	fnCache map[string]FunctionProto,
) (engine.NormalSuffixOperate, specialCase) {
	switch opt := op.(type) {
	default:
		panic(fmt.Errorf("cannot convert suffix operate %T:%v", op, op))
	case lower_ast.LiterialExpression:
		value := object.BoxLit(opt.Value)
		switch opt.Value {
		case true:
			return litTrue, specialCase{
				IsSpecial: true, IsLit: true, Lit: object.True,
			}
		case false:
			return litFalse, specialCase{
				IsSpecial: true, IsLit: true, Lit: object.False,
			}
		case nil:
			return litNil, specialCase{
				IsSpecial: true, IsLit: true, Lit: object.Nil,
			}
		default:
			return func(e engine.Env) object.Box {
					return value
				}, specialCase{
					IsSpecial: true, IsLit: true, Lit: value,
				}
		}

	case lower_ast.IdentifierExpression:
		switch opt.Scope {
		default:
			panic("unknown operate scope")
		case ast.IdentifierScopeLocal, ast.IdentifierScopeFree:
			slotIdx := localAndFreeLookup[opt]
			return func(e engine.Env) object.Box {
				slot := e.LocalAndFreeVars[slotIdx]
				if slot.Ref != nil {
					return slot.Ref.Value
				}
				return slot.Value
			}, specialCase{IsSpecial: true, IsLocalAndFreeSlot: true, SlotIdx: slotIdx}
		case ast.IdentifierScopeGlobal, ast.IdentifierScopeImport:
			slotIdx := globalAndImportLookup[opt]
			return func(e engine.Env) object.Box {
				return e.GlobalsAndImport[slotIdx]
			}, specialCase{}
		}
	case lower_ast.PrefixExpression:
		return compilePrefixExpression(opt.Operator, valueStack)
	case lower_ast.InfixExpression:
		return compileInfixExpression(opt.Operator, valueStack)
	case lower_ast.ListExpression:
		operands := valueStack.TakeN(int(opt.ElementsCount))
		ops := []engine.NormalSuffixOperate{}
		for _, op := range operands {
			ops = append(ops, op.op)
		}
		return func(e engine.Env) object.Box {
			values := object.NewList()
			for _, operand := range ops {
				values.AppendItem(operand(e))
			}
			return object.BoxList(values)
		}, specialCase{}
	case lower_ast.MapExpression:
		operands := valueStack.TakeN(int(opt.ConsumeOperand()))
		ops := []engine.NormalSuffixOperate{}
		for _, op := range operands {
			ops = append(ops, op.op)
		}
		return func(e engine.Env) object.Box {
			values := object.NewMap()
			for i := 0; i < len(ops); i += 2 {
				key := ops[i](e)
				value := ops[i+1](e)
				values.SetKeyValue(key, value)
			}
			return object.BoxMap(values)
		}, specialCase{}
	case lower_ast.AssignmentIdentifierExpression:
		value := valueStack.TakeN(1)[0].op
		lookupKey := lower_ast.IdentifierExpression{IdentifierName: opt.Identifier.Value, Scope: opt.Scope}
		switch opt.Scope {
		default:
			panic("unknown operate scope")
		case ast.IdentifierScopeLocal, ast.IdentifierScopeFree:
			slotIdx := localAndFreeLookup[lookupKey]
			return func(e engine.Env) object.Box {
				v := value(e)
				slot := e.LocalAndFreeVars[slotIdx]
				if slot.Ref != nil {
					slot.Ref.Value = v
					return v
				}
				e.LocalAndFreeVars[slotIdx].Value = v
				return v
			}, specialCase{}
		case ast.IdentifierScopeGlobal, ast.IdentifierScopeImport:
			slotIdx := globalAndImportLookup[lookupKey]
			return func(e engine.Env) object.Box {
				v := value(e)
				e.GlobalsAndImport[slotIdx] = v
				return v
			}, specialCase{}
		}
	case lower_ast.AssignmentAttributeExpression:
		operands := valueStack.TakeN(2)
		target, value := operands[0].op, operands[1].op
		attr := opt.Attribute
		return func(e engine.Env) object.Box {
			targetValue := target(e)
			assignValue := value(e)
			object.SetArrtibute(targetValue, attr, assignValue)
			return assignValue
		}, specialCase{}
	case lower_ast.AssignmentIndexExpression:
		operands := valueStack.TakeN(3)
		target, index, value := operands[0].op, operands[1].op, operands[2].op
		return func(e engine.Env) object.Box {
			targetValue := target(e)
			indexValue := index(e)
			assignValue := value(e)
			object.SetItem(targetValue, indexValue, assignValue)
			return assignValue
		}, specialCase{}
	case lower_ast.CreateEnclosureExpression:
		// 由于每个函数都要编译两遍，如果 ast 树中的子函数也要被编译两遍，那么最后会产生非常多重复的编译
		// 所以对于重复的函数片段，我们直接读取缓存
		compiledFp, ok := fnCache[strconv.Itoa(int(opt.FunctionProto.Hash()))]
		if !ok {
			// 如果确实没有编译，那么再编译
			compiledFp = CompileFnProtoWithCfg(opt.FunctionProto, localAndFreeLookup, globalAndImportLookup, fnCache)
			fnCache[strconv.Itoa(int(opt.FunctionProto.Hash()))] = compiledFp
		}
		return func(e engine.Env) object.Box {
			// 本质上，闭包就是将函数原型和捕获的自由变量进行封装
			captures := make([]object.RefOrValue, compiledFp.FreeCount)
			for localSlot, outerSlot := range compiledFp.FreeMapping {
				slot := e.LocalAndFreeVars[outerSlot]
				if slot.Ref == nil {
					slot.Ref = &object.Ref{Value: slot.Value}
					slot.Value = object.Empty
					e.LocalAndFreeVars[outerSlot] = slot
				}
				captures[localSlot] = slot
			}
			return object.BoxCustom(compiledFp.ToEnclosure(e.GlobalsAndImport, captures))
		}, specialCase{}
	}
}
