package compile

import (
	"fmt"
	ast "xex/ast"
	"xex/engine"
	lower_ast "xex/lower-ast"
	"xex/object"
)

type FunctionProto struct {
	// 代表这个函数参数数量
	ParametersCount int32
	// 代表这个函数局部变量(参数也算作局部变量的一部分)和Free变量总的槽位数
	// Parameters 在最前面
	LocalCount int32
	FreeCount  int32
	// 代表这个函数闭包从这个 functionProto 被构造时，外部的 LocalFree -> 闭包 Free 的映射关系
	// list 记录的是按顺序，从第一个到最后一个 Free 对应的外部的 LocalAndFree 的 Slot 号
	FreeMapping []int32

	// 代表这个函数从CFG转换而来的混合操作队列
	// 但是这里假设不是异步函数
	OperationsNonAsync []engine.HybirdOperation
	// 代表在同步模式下这个函数需要的最大辅助栈大小
	AuxStackSizeNonAsync int32
	// 但是这里假设是异步函数
	OperationsAsync []engine.HybirdOperation
	// 代表在异步模式下这个函数需要的最大辅助栈大小
	AuxStackSizeAsync int32
	NonAsync          bool
}

type Lookup map[lower_ast.IdentifierExpression]int

func (l Lookup) ToSlots(m map[lower_ast.IdentifierExpression]any) []object.Box {
	slots := make([]object.Box, len(l))
	for k, v := range m {
		slots[l[k]] = object.BoxAny(v)
	}
	return slots
}

func (compiledFp FunctionProto) ToEnclosure(GlobalsAndImport []object.Box, Captures []object.RefOrValue) *engine.Enclosure {
	return &engine.Enclosure{
		ParametersCount:      int(compiledFp.ParametersCount),
		LocalCount:           int(compiledFp.LocalCount),
		LocalAndFreeCount:    int(compiledFp.LocalCount + compiledFp.FreeCount),
		GlobalsAndImport:     GlobalsAndImport,
		Captures:             Captures,
		AuxStackSizeNonAsync: int(compiledFp.AuxStackSizeNonAsync),
		OperationsNonAsync:   compiledFp.OperationsNonAsync,
		AuxStackSizeAsync:    int(compiledFp.AuxStackSizeAsync),
		OperationsAsync:      compiledFp.OperationsAsync,
		NonAsync:             compiledFp.NonAsync,
	}
}

func CompileFnProto(
	proto lower_ast.FunctionProto,
	globals []string,
	fnCache map[string]FunctionProto,
) (compiledFp FunctionProto, globalAndImportLookup Lookup) {
	if fnCache == nil {
		fnCache = map[string]FunctionProto{}
	}
	allImports := proto.GetAllImports()
	globalAndImportLookup = genIdentifierScopsLookup([]IdentiferNamesWithScope{IdentiferNamesWithScope{
		symbolNames: globals,
		scopeName:   ast.IdentifierScopeGlobal,
		noSort:      true,
	}, IdentiferNamesWithScope{
		symbolNames: allImports,
		scopeName:   ast.IdentifierScopeImport,
		noSort:      true,
	}})
	fp := CompileFnProtoWithCfg(proto, Lookup{}, globalAndImportLookup, fnCache)
	return fp, globalAndImportLookup
}

func CompileFnProtoWithCfg(
	fnProto lower_ast.FunctionProto,
	localAndFreeLookup Lookup,
	globalAndImportLookup Lookup,
	fnCache map[string]FunctionProto,
) FunctionProto {
	exist := map[string]struct{}{}
	localSymbolsNames := []string{}
	freeSymbolsNames := []string{}
	// 确保参数顶在最前面
	for _, ls := range fnProto.ParameterSymbols {
		localSymbolsNames = append(localSymbolsNames, ls.Value)
		exist[ls.Value] = struct{}{}
	}
	// 然后是其他局部变量
	for _, ls := range fnProto.LocalSymbols {
		if _, ok := exist[ls.Value]; ok {
			continue
		}
		localSymbolsNames = append(localSymbolsNames, ls.Value)
	}
	if len(localSymbolsNames) != len(fnProto.LocalSymbols) {
		panic(fmt.Errorf("parameters: %v not fully included in local symbols: %v", fnProto.ParameterSymbols, fnProto.LocalSymbols))
	}
	// 最后才是自由变量
	for _, ls := range fnProto.FreeSymbols {
		freeSymbolsNames = append(freeSymbolsNames, ls.Value)
	}
	// 生成新的，内部的局部和自由变量布局
	innerLookup := genIdentifierScopsLookup([]IdentiferNamesWithScope{
		{symbolNames: localSymbolsNames, scopeName: ast.IdentifierScopeLocal, noSort: true},
		{symbolNames: freeSymbolsNames, scopeName: ast.IdentifierScopeFree, noSort: true},
	})
	freeMapping := []int32{}
	for i, freeName := range freeSymbolsNames {
		// 先试试在不在外部的局部变量中
		outerSlotIdx, ok := localAndFreeLookup[lower_ast.IdentifierExpression{IdentifierName: freeName, Scope: ast.IdentifierScopeLocal}]
		if !ok {
			// 不在就试试 Free 变量
			outerSlotIdx, ok = localAndFreeLookup[lower_ast.IdentifierExpression{IdentifierName: freeName, Scope: ast.IdentifierScopeFree}]
			if !ok {
				// 找不到这个变量了
				panic(fmt.Errorf("%v not found in both outer local and free: %v", freeName, localAndFreeLookup))
			}
		}
		innerSlot := innerLookup[lower_ast.IdentifierExpression{IdentifierName: freeName, Scope: ast.IdentifierScopeFree}]
		if innerSlot != i+len(localSymbolsNames) {
			panic("free slot index mismatch")
		}
		freeMapping = append(freeMapping, int32(outerSlotIdx))
	}
	// 由于每个函数都要编译两遍，如果 ast 树中的子函数也要被编译两遍，那么最后会产生非常多重复的编译。
	// 整棵函数树共享同一个 cache，确保每个嵌套函数只编译一次。
	nonAsyncHops := CompileLowerAst(fnProto.Graph, innerLookup, globalAndImportLookup, false, fnCache)
	asyncHops := CompileLowerAst(fnProto.Graph, innerLookup, globalAndImportLookup, true, fnCache)
	fp := FunctionProto{
		ParametersCount:      int32(len(fnProto.ParameterSymbols)),
		LocalCount:           int32(len(fnProto.LocalSymbols)),
		FreeCount:            int32(len(fnProto.FreeSymbols)),
		FreeMapping:          freeMapping,
		AuxStackSizeNonAsync: int32(nonAsyncHops.auxSlotSize),
		OperationsNonAsync:   nonAsyncHops.operations,
		AuxStackSizeAsync:    int32(asyncHops.auxSlotSize),
		OperationsAsync:      asyncHops.operations,
		NonAsync:             fnProto.NonAsync,
	}
	return fp
}
