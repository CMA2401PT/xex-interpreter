package engine

import (
	"fmt"
	"xex/object"
)

type Enclosure struct {
	// 代表这个函数参数数量
	ParametersCount int
	// 代表这个函数局部变量(参数也算作局部变量的一部分)和Free变量总的槽位数
	// Parameters 在最前面
	LocalCount        int
	LocalAndFreeCount int
	// 代表在同步模式下这个函数需要的最大辅助栈大小
	AuxStackSizeNonAsync int
	// 代表在异步模式下这个函数需要的最大辅助栈大小
	AuxStackSizeAsync int
	// 从外部捕获的 Free 变量
	Captures []object.RefOrValue
	// 和外部共享的 GlobalsAndImport
	GlobalsAndImport []object.Box
	// 代表这个函数从CFG转换而来的混合操作队列
	// 但是这里假设不是异步函数
	OperationsNonAsync []HybirdOperation
	// 但是这里假设是异步函数
	OperationsAsync []HybirdOperation
	NonAsync        bool
}

func syncRunCallable(e *Enclosure, args []object.Box) object.Box {
	if e.ParametersCount != len(args) {
		panic(fmt.Errorf("expect %v args, but get %v args", e.ParametersCount, len(args)))
	}
	localAndFrees := append(make([]object.RefOrValue, e.LocalCount), e.Captures...)
	auxSlot := make([]object.Box, e.AuxStackSizeNonAsync)
	for i := range e.LocalCount {
		if i < e.ParametersCount {
			localAndFrees[i] = object.RefOrValue{Value: args[i]}
		}
	}
	ne := newNonAsyncEvaluator()
	env := Env{
		GlobalsAndImport:  e.GlobalsAndImport,
		LocalAndFreeVars:  localAndFrees,
		AuxSlots:          auxSlot,
		NonAsyncEvaluator: ne,
	}

	ops := e.OperationsNonAsync
	ret := ne.run(env, ops)
	return ret
}

// GetCallable 本质上是假设这个是普通函数，自己且底层函数都是普通函数
func (e *Enclosure) GetCallable() object.NormalHostFn {
	return func(args []object.Box) object.Box {
		return syncRunCallable(e, args)
	}
}

// GetCallable 本质上是假设这个是异步函数，自己且底层函数都可能是异步函数
func (e *Enclosure) GetAsyncCallable() object.AsyncHostFn {
	return func(handle object.AsyncHandleType, args []object.Box) object.AsyncYieldReason {
		return EvalEnclosureAsync(e, handle, args)
	}
}

// 相当于 js 的 this+函数或者 python 的 self + 函数
// 使用时将 Self 插入到第一个参数
type EncolsureBind struct {
	Self object.Box
	Enc  *Enclosure
}

func (e EncolsureBind) GetCallable() object.NormalHostFn {
	return func(args []object.Box) object.Box {
		return syncRunCallable(e.Enc, append([]object.Box{e.Self}, args...))
	}
}

// GetCallable 本质上是假设这个是异步函数，自己且底层函数都可能是异步函数
func (e EncolsureBind) GetAsyncCallable() object.AsyncHostFn {
	return func(handle object.AsyncHandleType, args []object.Box) object.AsyncYieldReason {
		return EvalEnclosureAsync(e.Enc, handle, append([]object.Box{e.Self}, args...))
	}
}
