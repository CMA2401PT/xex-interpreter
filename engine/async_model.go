package engine

import (
	"fmt"
	"xex/async"
	lower_ast "xex/lower-ast"
	"xex/object"
)

type frame struct {
	pc               int
	resultSlot       int
	ops              []HybirdOperation
	LocalAndFreeVars []object.RefOrValue
	AuxSlots         []object.Box
}

func EvalEnclosureAsync(e *Enclosure, handle object.AsyncHandleType, args []object.Box) object.AsyncYieldReason {
	if e.ParametersCount != len(args) {
		panic(fmt.Errorf("expect %v args, but get %v args", e.ParametersCount, len(args)))
	}

	frames := []frame{}

	resueLocalSlots := []object.RefOrValue{}
	localSlotsUsed := 0
	reuseAuxSlots := []object.Box{}
	auxSlotsUsed := 0

	takeLocalSlot := func(size int) []object.RefOrValue {
		s := localSlotsUsed
		e := localSlotsUsed + size
		if e >= len(resueLocalSlots) {
			resueLocalSlots = append(resueLocalSlots, make([]object.RefOrValue, e-len(resueLocalSlots))...)
		}
		slots := resueLocalSlots[s:e]
		localSlotsUsed = e
		for i := range slots {
			slots[i] = object.RefOrValue{}
		}
		return slots
	}

	takeAuxSlot := func(size int) []object.Box {
		s := auxSlotsUsed
		e := auxSlotsUsed + size
		if e >= len(reuseAuxSlots) {
			reuseAuxSlots = append(reuseAuxSlots, make([]object.Box, e-len(reuseAuxSlots))...)
		}
		slots := reuseAuxSlots[s:e]
		auxSlotsUsed = e
		for i := range slots {
			slots[i] = object.Empty
		}
		return slots
	}

	putLocalSlot := func(size int) {
		localSlotsUsed -= size
	}

	putAuxSlot := func(size int) {
		auxSlotsUsed -= size
	}

	var ops []HybirdOperation
	var pc int
	var env Env
	env.GlobalsAndImport = e.GlobalsAndImport

	updateEnv := func(e *Enclosure) {
		var auxSlot []object.Box
		var localAndFrees []object.RefOrValue
		localAndFreeSize := int(e.LocalAndFreeCount)
		if localAndFreeSize > 0 {
			localAndFrees = takeLocalSlot(localAndFreeSize)
		}
		if e.AuxStackSizeAsync > 0 {
			auxSlot = takeAuxSlot(int(e.AuxStackSizeAsync))
		}
		for offset, c := range e.Captures {
			localAndFrees[offset+int(e.LocalCount)] = c
		}
		env.AuxSlots = auxSlot
		env.LocalAndFreeVars = localAndFrees
	}

	ops = e.OperationsAsync
	pc = 0
	updateEnv(e)
	for i, v := range args {
		env.LocalAndFreeVars[i] = object.RefOrValue{Value: v}
	}
	var run func(handle object.AsyncHandleType) object.AsyncYieldReason
	run = func(handle object.AsyncHandleType) object.AsyncYieldReason {
		for {
			for {
				op := &ops[pc]
				switch op.OperationType {
				default:
					panic(fmt.Errorf("unexpected operation type %v at pc=%d", op.OperationType, pc))
				case OpTypeGraphAndKeep:
					env.AuxSlots[op.ResultSlot] = op.Graph(env)
					pc += 1
				case OpTypeGraphAndDrop:
					op.Graph(env)
					pc += 1
				case OpTypeJumpTrue:
					if op.Graph(env) == object.True {
						pc = int(op.JumpTo)
					} else {
						pc += 1
					}
				case OpTypeJumpNoCond:
					if op.JumpTo != lower_ast.CFGReturnTarget {
						if op.Graph != nil {
							op.Graph(env)
						}
						pc = int(op.JumpTo)
					}
					ret := op.Lit
					if op.Graph != nil {
						ret = op.Graph(env)
					}
					// 栈帧深度为0，已经完成
					if len(frames) == 0 {
						return async.YieldByFinish[[]object.Box, object.Box]{ret}
					}
					// 完成调用，退回上一帧，返还资源
					// 归还资源
					localAndFreeSize := len(env.LocalAndFreeVars)
					if localAndFreeSize > 0 {
						putLocalSlot(localAndFreeSize)
					}
					auxSize := len(env.AuxSlots)
					if auxSize > 0 {
						putAuxSlot(auxSize)
					}
					frame := frames[len(frames)-1]
					frames = frames[:len(frames)-1]
					ops = frame.ops
					pc = frame.pc
					// 还原 env
					env.LocalAndFreeVars = frame.LocalAndFreeVars
					env.AuxSlots = frame.AuxSlots
					// 结果压入辅助栈中
					env.AuxSlots[frame.resultSlot] = ret
				case OpTypeYield:
					slot := op.ResultSlot
					reason := object.UnBoxCustomType[async.CanCreateYieldReason[[]object.Box, object.Box]](env.AuxSlots[slot])
					pc += 1

					return reason.SuspendAndYield(handle.Resume, func(newHandle object.AsyncHandleType, args []object.Box) object.AsyncYieldReason {
						env.AuxSlots[slot] = object.BoxAny(args)
						return run(newHandle)
					})
				// 	panic(fmt.Errorf("unexpected async operation type %v at pc=%d in non-async callable", op.OperationType, pc))
				case OpTypeCall:
					fn := env.AuxSlots[op.FnSlot]
					resultSlot := op.FnSlot
					if fn.BasicType == object.BasicTypEnclosure {
						e = object.UnBoxObjEnclosure[*Enclosure](fn)
						if e.NonAsync {
							engineuator := newNonAsyncEvaluator()
							engineuator.auxSlotsUsed = auxSlotsUsed
							engineuator.reuseAuxSlots = reuseAuxSlots
							engineuator.localSlotsUsed = localSlotsUsed
							engineuator.resueLocalSlots = resueLocalSlots
							newEnv := Env{
								GlobalsAndImport:  env.GlobalsAndImport,
								NonAsyncEvaluator: engineuator,
							}
							aops := []NormalSuffixOperate{}
							for i := range op.ArgsEnd - op.ArgsStart {
								v := env.AuxSlots[i+op.ArgsStart]
								aops = append(aops, func(e Env) object.Box {
									return v
								})
							}
							env.AuxSlots[resultSlot] = engineuator.Eval(e, aops, newEnv)
							pc += 1
							continue
						}

						oldEnv := env
						// 保存当前帧
						frames = append(frames, frame{
							pc:               pc + 1, // 返回时执行下一个指令
							resultSlot:       int(resultSlot),
							ops:              ops,
							LocalAndFreeVars: oldEnv.LocalAndFreeVars,
							AuxSlots:         oldEnv.AuxSlots,
						})
						// 转到新的帧
						updateEnv(e)
						for i := range op.ArgsEnd - op.ArgsStart {
							env.LocalAndFreeVars[i] = object.RefOrValue{Value: oldEnv.AuxSlots[i+op.ArgsStart]}
						}
						ops = e.OperationsAsync
						pc = 0
					} else {
						args := env.AuxSlots[op.ArgsStart:op.ArgsEnd]
						asyncFn, ok := object.TryGetAsyncCallable(fn)
						if ok {
							taskDelegate := async.YieldByAwait[[]object.Box, object.Box]{
								Target: asyncFn,
								Args:   args,
								Shim:   func(ret object.Box) []object.Box { return []object.Box{ret} },
							}
							pc += 1
							return taskDelegate.SuspendAndYield(handle.Resume, func(newHandle object.AsyncHandleType, args []object.Box) object.AsyncYieldReason {
								env.AuxSlots[resultSlot] = args[0]
								return run(newHandle)
							})
						}
						env.AuxSlots[resultSlot] = object.GetCallable(fn)(args)
						pc += 1
						continue
					}
				}
			}
		}
	}
	return run(handle)
}
