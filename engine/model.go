package engine

import (
	"fmt"
	lower_ast "xex/lower-ast"
	"xex/object"
)

// 我们已经知道
// lower_ast.SuffixOperatesSequence 最后一定生成一个值
// 中间使用 nil 分隔并吃掉多余的值
// 现在关键的问题是，遇到 yield 还有 函数该怎么办？
// 特殊动作指的是 Yield 和 Call
// 编译为图结构可以很好的利用高级语言本身的特性加速
// 但是这样显然无法处理挂起的模型
// 构造一个混合操作队列，这个队列每个动作都为求一个子图，每个子图只会产生一个值，
// 或执行一次特殊动作
// 维护一个辅助值队列，用以桥接子图和特殊动作
// 特殊动作通过辅助值队列读取子图的值作为参数
// 子图通过辅助值队列读取特殊操作的值作为参数
type HybirdOperation struct {
	OperationType HybirdOperationType

	// 当 OpTypeCall 时
	// 函数位于辅助栈的槽位 和需要的参数数量，参数位于辅助栈的 [ArgsStart:ArgsEnd]
	FnSlot, ArgsStart, ArgsEnd uint8

	// 当 OpTypeGraph / OpTypeYield 时, 将其值放在辅助栈的哪个槽
	// 当 OpTypeCall 时, ResultSlot 与 FnSlot 相同
	ResultSlot uint8

	// 当 OpTypeJump 时，跳转或为真时跳转目标
	// 为 -1 时代表函数结束
	JumpTo int32

	// 当 OpTypeGraph / OpTypeJump 时, 子图求值函数的根节点
	Graph NormalSuffixOperate

	// 当 OpTypeJumpNoCond 且 JumpTo=-1 时，在 Graph=nil 时返回该值
	Lit object.Box
}

type HybirdOperationType uint8

const (
	// 对 Graph 求值并保留值到辅助栈的 ResultSlot
	OpTypeGraphAndKeep = HybirdOperationType(iota)
	// 对 Graph 求值并丢弃值，只使用其副作用
	OpTypeGraphAndDrop
	// 函数调用，并将值保留到 ResultSlot
	OpTypeCall
	// Yield 函数，其返回 ResultSlot 到执行器之外，并将外部给的值放回原位
	// 也就是替换 ResultSlot 的值
	OpTypeYield
	// 无条件跳转到某个地方，最后一个操作进行但值不再写入辅助栈
	OpTypeJumpNoCond
	// 有条件跳转到某个地方，最后一个操作进行但值不再写入辅助栈，
	// 值必须为 bool 类型
	// 为 True 时跳转到 JumpTo，否则继续执行下一条指令
	OpTypeJumpTrue
)

func NewHybirdOperationGraphKeep(graph NormalSuffixOperate, slot int) HybirdOperation {
	return HybirdOperation{
		OperationType: OpTypeGraphAndKeep, Graph: graph, ResultSlot: uint8(slot),
	}
}

func NewHybirdOperationGraphDrop(graph NormalSuffixOperate) HybirdOperation {
	return HybirdOperation{
		OperationType: OpTypeGraphAndDrop, Graph: graph,
	}
}

func NewHybirdOperationCall(fnSlot, argsCount int) HybirdOperation {
	return HybirdOperation{
		OperationType: OpTypeCall,
		FnSlot:        uint8(fnSlot), ArgsStart: uint8(fnSlot + 1), ArgsEnd: uint8(fnSlot + 1 + argsCount),
		// 其结果和 fn 应该位于同一个槽位
		ResultSlot: uint8(fnSlot),
	}
}

func NewHybirdOperationYield(slot int) HybirdOperation {
	return HybirdOperation{
		OperationType: OpTypeYield,
		ResultSlot:    uint8(slot),
	}
}

func NewHybirdOperationJumpNoCond(target int32, graph NormalSuffixOperate) HybirdOperation {
	return HybirdOperation{
		OperationType: OpTypeJumpNoCond,
		JumpTo:        target,
		Graph:         graph,
	}
}

func NewHybirdOperationJumpNoCondWithLit(target int32, lit object.Box) HybirdOperation {
	return HybirdOperation{
		OperationType: OpTypeJumpNoCond,
		JumpTo:        target,
		Lit:           lit,
	}
}

func NewHybirdOperationJumpTrue(target int32, graph NormalSuffixOperate) HybirdOperation {
	return HybirdOperation{
		OperationType: OpTypeJumpTrue,
		JumpTo:        target,
		Graph:         graph,
	}
}

// 所有后缀表达式，除了 Yield 和 Call
// (因为被Call的对象内部可能有 yield，但是在遇到 yield 之前我们都是不知道的)
// 都可以表示为对 Env 的直接操作而没有其他副作用
type NormalSuffixOperate = func(env Env) object.Box

type Env struct {
	// 所有的 Global 和 Import 值
	GlobalsAndImport []object.Box
	// 所有的 Local 和 Free 变量
	// Local 在前，Free 在后，Local 开头是参数
	LocalAndFreeVars []object.RefOrValue // 这里是引用
	// 用于后缀表达式的辅助栈，只有退化值和桥接操作需要用到这个栈
	AuxSlots []object.Box
	// 用于在函数调用时复用部分信息以节约内存，减少分配
	*NonAsyncEvaluator
}

type SlotProvider struct {
	resueLocalSlots []object.RefOrValue
	localSlotsUsed  int
	reuseAuxSlots   []object.Box
	auxSlotsUsed    int
}

type NonAsyncEvaluator struct {
	SlotProvider
}

func (ne *SlotProvider) takeLocalSlot(size int) []object.RefOrValue {
	s := ne.localSlotsUsed
	e := ne.localSlotsUsed + size
	if e >= len(ne.resueLocalSlots) {
		ne.resueLocalSlots = append(ne.resueLocalSlots, make([]object.RefOrValue, e-len(ne.resueLocalSlots))...)
	}
	slots := ne.resueLocalSlots[s:e]
	ne.localSlotsUsed = e
	for i := range slots {
		slots[i] = object.RefOrValue{}
	}
	return slots
}

func (ne *SlotProvider) putLocalSlot(size int) {
	ne.localSlotsUsed -= size
}

func (ne *SlotProvider) takeAuxSlot(size int) []object.Box {
	s := ne.auxSlotsUsed
	e := ne.auxSlotsUsed + size
	if e >= len(ne.reuseAuxSlots) {
		ne.reuseAuxSlots = append(ne.reuseAuxSlots, make([]object.Box, e-len(ne.reuseAuxSlots))...)
	}
	slots := ne.reuseAuxSlots[s:e]
	ne.auxSlotsUsed = e
	for i := range slots {
		slots[i] = object.Empty
	}
	return slots
}

func (ne *SlotProvider) putAuxSlot(size int) {
	ne.auxSlotsUsed -= size
}

func newSlotProvider() *SlotProvider {
	return &SlotProvider{
		resueLocalSlots: make([]object.RefOrValue, 0),
		reuseAuxSlots:   make([]object.Box, 0),
	}
}

func newNonAsyncEvaluator() *NonAsyncEvaluator {
	return &NonAsyncEvaluator{
		SlotProvider: *newSlotProvider(),
	}
}

func (ne *NonAsyncEvaluator) Eval(e *Enclosure, args []NormalSuffixOperate, outerE Env) object.Box {
	if e.ParametersCount != len(args) {
		panic(fmt.Errorf("expect %v args, but get %v args", e.ParametersCount, len(args)))
	}
	var auxSlot []object.Box
	var localAndFrees []object.RefOrValue
	localAndFreeSize := int(e.LocalAndFreeCount)
	if localAndFreeSize > 0 {
		localAndFrees = ne.takeLocalSlot(localAndFreeSize)
		defer ne.putLocalSlot(localAndFreeSize)
	}
	if e.AuxStackSizeNonAsync > 0 {
		auxSlot = ne.takeAuxSlot(int(e.AuxStackSizeNonAsync))
		defer ne.putAuxSlot(int(e.AuxStackSizeNonAsync))
	}
	// append(make([]*object.Cell, e.LocalCount), e.Captures...)
	for i := range e.ParametersCount {
		localAndFrees[i] = object.RefOrValue{Value: args[i](outerE)}
	}
	for offset, c := range e.Captures {
		localAndFrees[offset+int(e.LocalCount)] = c
	}
	outerE.LocalAndFreeVars = localAndFrees
	outerE.AuxSlots = auxSlot
	ops := e.OperationsNonAsync
	ret := ne.run(outerE, ops)
	return ret
}

func (ne *NonAsyncEvaluator) EvalWithThis(e *Enclosure, this object.Box, args []NormalSuffixOperate, outerE Env) object.Box {
	if e.ParametersCount != len(args)+1 {
		panic(fmt.Errorf("expect %v args, but get %v+1 args", e.ParametersCount, len(args)))
	}
	var auxSlot []object.Box
	var localAndFrees []object.RefOrValue
	localAndFreeSize := int(e.LocalAndFreeCount)
	if localAndFreeSize > 0 {
		localAndFrees = ne.takeLocalSlot(localAndFreeSize)
		defer ne.putLocalSlot(localAndFreeSize)
	}
	if e.AuxStackSizeNonAsync > 0 {
		auxSlot = ne.takeAuxSlot(int(e.AuxStackSizeNonAsync))
		defer ne.putAuxSlot(int(e.AuxStackSizeNonAsync))
	}
	// 插入 self/this
	localAndFrees[0] = object.RefOrValue{Value: this}
	for i := range e.ParametersCount {
		localAndFrees[i+1] = object.RefOrValue{Value: args[i](outerE)}
	}
	for offset, c := range e.Captures {
		localAndFrees[offset+int(e.LocalCount)] = c
	}
	outerE.LocalAndFreeVars = localAndFrees
	outerE.AuxSlots = auxSlot
	ops := e.OperationsNonAsync
	ret := ne.run(outerE, ops)
	return ret
}

func (NonAsyncEvaluator) run(env Env, ops []HybirdOperation) object.Box {
	pc := 0
	for {
		op := &ops[pc]
		switch op.OperationType {
		default:
			panic(fmt.Errorf("unexpected operation type %v at pc=%d", op.OperationType, pc))
		// case OpTypeCall, OpTypeYield:
		// 	panic(fmt.Errorf("unexpected async operation type %v at pc=%d in non-async callable", op.OperationType, pc))
		case OpTypeGraphAndKeep:
			env.AuxSlots[op.ResultSlot] = op.Graph(env)
			pc += 1
		case OpTypeGraphAndDrop:
			op.Graph(env)
			pc += 1
		case OpTypeJumpNoCond:
			if op.JumpTo == lower_ast.CFGReturnTarget {
				if op.Graph == nil {
					return op.Lit
				}
				ret := op.Graph(env)
				return ret
			}
			if op.Graph != nil {
				op.Graph(env)
			}
			pc = int(op.JumpTo)
		case OpTypeJumpTrue:
			if op.Graph(env) == object.True {
				pc = int(op.JumpTo)
			} else {
				pc += 1
			}
		}
	}
}
