package compile

import (
	"slices"
	"xex/engine"
	lower_ast "xex/lower-ast"
	"xex/object"
)

type compileValue struct {
	// 进行对这个 op 进行求值，即可获得对应位置栈应有的值
	op engine.NormalSuffixOperate
	// isFallback 判断生成这个值的 op 是不是已经放入混合操作栈了
	// 也就是说，对应的值已经一定会生成了
	// 在这种情况下，op 只是从辅助栈中再次取这个值
	// 如果是的话，无需对这个再次进行 fallback 了
	isFallback bool
}

type specialCase struct {
	IsSpecial          bool
	IsLocalAndFreeSlot bool
	SlotIdx            int
	IsLit              bool
	Lit                object.Box
}

type compileValueWithSpec struct {
	compileValue
	specialCase
}

type compileValueStack struct {
	valueBySuffixOperates []compileValue
	specialCases          []specialCase
	ptr                   int
}

func (sv *compileValueStack) TakeN(n int) []compileValueWithSpec {
	vs := sv.valueBySuffixOperates[sv.ptr-n : sv.ptr]
	sc := sv.specialCases[sv.ptr-n : sv.ptr]
	sv.ptr -= n
	out := make([]compileValueWithSpec, len(vs))
	for i, v := range vs {
		out[i] = compileValueWithSpec{compileValue: v, specialCase: sc[i]}
	}
	return out
}

func (sv *compileValueStack) TakeAndCopyRest() (
	[]compileValue, []specialCase,
) {
	vs := sv.valueBySuffixOperates[:sv.ptr]
	sc := sv.specialCases[:sv.ptr]
	sv.valueBySuffixOperates = nil
	sv.specialCases = nil
	sv.ptr = 0
	return slices.Clone(vs), slices.Clone(sc)
}

func (sv *compileValueStack) Put(op compileValue) {
	if len(sv.valueBySuffixOperates) == sv.ptr {
		sv.valueBySuffixOperates = append(sv.valueBySuffixOperates, op)
		sv.specialCases = append(sv.specialCases, specialCase{})
		sv.ptr += 1
		return
	} else {
		sv.valueBySuffixOperates[sv.ptr] = op
		sv.specialCases[sv.ptr] = specialCase{}
		sv.ptr += 1
	}
}

func (sv *compileValueStack) PutWithSpecial(
	op compileValue, sc specialCase,
) {
	if len(sv.valueBySuffixOperates) == sv.ptr {
		sv.valueBySuffixOperates = append(sv.valueBySuffixOperates, op)
		sv.specialCases = append(sv.specialCases, sc)
		sv.ptr += 1
		return
	} else {
		sv.valueBySuffixOperates[sv.ptr] = op
		sv.specialCases[sv.ptr] = sc
		sv.ptr += 1
	}
}

func Compile(ast any,
	localAndFreeLookup Lookup,
	globalAndImportLookup Lookup,
	allowAsync bool,
	fnCache map[string]FunctionProto,
) any {
	switch astt := ast.(type) {
	default:
		panic("not implement error")
	case lower_ast.CFGBlock[lower_ast.SuffixOperatesSequence]:
		return CompileCFGBlock(astt, localAndFreeLookup, globalAndImportLookup, allowAsync, fnCache)
	case lower_ast.LowerAst:
		return CompileLowerAst(astt, localAndFreeLookup, globalAndImportLookup, allowAsync, fnCache)
	case lower_ast.SuffixOperatesSequence:
		return CompileSeq(astt, localAndFreeLookup, globalAndImportLookup, allowAsync, fnCache)
	case lower_ast.FunctionProto:
		return CompileFnProtoWithCfg(astt, localAndFreeLookup, globalAndImportLookup, fnCache)
	}
}
