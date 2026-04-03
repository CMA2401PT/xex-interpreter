package async

import (
	"sync/atomic"
	"time"
)

var loopIdx atomic.Int32

// TaskAux 反应的不是当前的 Step，
// 而是多个 Step 构成的完整过程的附加数据， 这个过程才对应 Generator/async 函数
type TaskAux[ArgsT any, RetT any] struct {
	// 全局唯一 task 编号（不仅仅是这个 scheduler）
	uniqueIdx int32
	// 等待当前 Task 结果的所有回调
	resultFut           *Future[ArgsT, RetT]
	resultShim          func(ret RetT) ArgsT
	yieldTicketConsumed int
	yieldTicketCurrent  int // 使用这个计数器防止错误的重用同一个放回函数
}

// getYieldTicket 和 consumeYieldTicket 是一对守卫函数
// 为当前 yield 生成返回值的回执
func (task *TaskAux[ArgsT, RetT]) getYieldTicket() int {
	if task.yieldTicketCurrent != task.yieldTicketConsumed {
		panic("one yield ticket already created but not consumed")
	}
	task.yieldTicketCurrent += 1
	return task.yieldTicketCurrent
}

// 通过 yieldTicket 检查是否是当前需要的，唯一的 data=yield reason 的返回值
func (task *TaskAux[ArgsT, RetT]) consumeYieldTicket(yieldTicket int) {
	if yieldTicket != task.yieldTicketCurrent {
		panic("invalid yield ticket")
	}
	if yieldTicket == task.yieldTicketConsumed {
		panic("reuse of a consumed yield ticket")
	}
	task.yieldTicketConsumed = yieldTicket
}

// 检查当前yield 是否已经获得返回值
func (task *TaskAux[ArgsT, RetT]) yieldAlreadyReceivedValue() bool {
	return task.yieldTicketConsumed == task.yieldTicketCurrent
}

// 代表一个 Task 就绪，
// Task 当前未在运行，
// Data, Ticket 则是驱动其运行的数据要素
// Next 则是当前 Task 即将运行的 Step
type Event[ArgsT any, RetT any] struct {
	// 当前多个 Step 构成的过程的附加数据
	Aux *TaskAux[ArgsT, RetT]
	// 下一个要运行的 Step
	Next Step[ArgsT, RetT]
	// 下一个要运行的 Step 的驱动数据
	Data ArgsT
}

type ExternEventReceiver[ArgsT any, RetT any] interface {
	ReceiveEvent(ev Event[ArgsT, RetT])
}

// 生成多个 Step 构成的完整过程的第一步
func NewReadyTask[ArgsT any, RetT any](step Step[ArgsT, RetT], args ArgsT) Event[ArgsT, RetT] {
	state := &TaskAux[ArgsT, RetT]{}
	return Event[ArgsT, RetT]{
		Aux:  state,
		Next: step,
		Data: args,
	}
}

type sleepingTask[ArgsT any, RetT any] struct {
	Next Step[ArgsT, RetT]
	// ReadyTask 的生成直接通过调用 Resume 实现
	Resume[ArgsT, RetT]
	// AwakeAt 指定其需要 Sleep 到的时间，可能小于等于现在的时间
	AwakeAt time.Time
}

type EventLoop[ArgsT any, RetT any] struct {
	// sleepingTask 是 sleep 的任务 heap，从而确保最先该被唤醒的总是在最前
	sleepingTasks       *Heap[sleepingTask[ArgsT, RetT]]
	events              []Event[ArgsT, RetT]
	unfinishTasks       int
	externEventReceiver ExternEventReceiver[ArgsT, RetT]
	loopIdx             int32
	coroIdxCount        int32
	runningCoroIdx      int32
}

func NewEventLoop[ArgsT any, RetT any](externEventReceiver ExternEventReceiver[ArgsT, RetT]) *EventLoop[ArgsT, RetT] {
	return &EventLoop[ArgsT, RetT]{
		sleepingTasks: newHeap(func(a, b sleepingTask[ArgsT, RetT]) (aPriorThanB bool) {
			return a.AwakeAt.Before(b.AwakeAt)
		}),
		events:              make([]Event[ArgsT, RetT], 0),
		unfinishTasks:       0,
		externEventReceiver: externEventReceiver,
		loopIdx:             loopIdx.Add(1),
		coroIdxCount:        1,
	}
}

type handle[ArgsT any, RetT any] struct {
	// resume 函数的唯一访问守卫
	yieldTicket int
	// 被设置的Loop本身
	loop *EventLoop[ArgsT, RetT]
	// 任务的辅助函数
	aux *TaskAux[ArgsT, RetT]
}

// 这个总是被loop框架外部函数使用，所以总是 coroSafe 的
func (h handle[ArgsT, RetT]) Resume(next Step[ArgsT, RetT], args ArgsT) {
	readyTask := Event[ArgsT, RetT]{
		Aux:  h.aux,
		Next: next,
		Data: args,
	}
	h.aux.consumeYieldTicket(h.yieldTicket)
	h.loop.externEventReceiver.ReceiveEvent(readyTask)

}

// 这个总是被框架使用，所以总是 coroSafe 的
func (h handle[ArgsT, RetT]) resumeNonCoroSafe(next Step[ArgsT, RetT], args ArgsT) {
	readyTask := Event[ArgsT, RetT]{
		Aux:  h.aux,
		Next: next,
		Data: args,
	}
	h.aux.consumeYieldTicket(h.yieldTicket)
	h.loop.events = append(h.loop.events, readyTask)
}

func (h handle[ArgsT, RetT]) CorotineId() int32 {
	return h.aux.uniqueIdx
}

func (h handle[ArgsT, RetT]) CreateTask(target Step[ArgsT, RetT], args ArgsT, shim func(RetT) ArgsT) *Future[ArgsT, RetT] {
	fut := &Future[ArgsT, RetT]{}
	isCoroSafe := h.aux.uniqueIdx == h.loop.runningCoroIdx
	h.loop.CreateAndAddTask(isCoroSafe, target, args, fut, shim)
	return fut
}

// 为 task 创建一个 resmue 函数
func (loop *EventLoop[ArgsT, RetT]) createHandle(aux *TaskAux[ArgsT, RetT]) handle[ArgsT, RetT] {
	// resume 函数的唯一访问守卫
	ticket := aux.getYieldTicket()
	return handle[ArgsT, RetT]{
		yieldTicket: ticket,
		loop:        loop,
		aux:         aux,
	}
}

// 若 loop 正在运行且从loop goruntine 之外添加 task，应当 coroSafe=false
func (loop *EventLoop[ArgsT, RetT]) addTask(coroSafe bool, task Event[ArgsT, RetT]) {
	coroIdx := loop.coroIdxCount
	coroIdx = coroIdx*1000 + loop.loopIdx
	loop.coroIdxCount += 1
	task.Aux.uniqueIdx = coroIdx
	loop.unfinishTasks += 1
	if coroSafe {
		loop.events = append(loop.events, task)
	} else {
		loop.externEventReceiver.ReceiveEvent(task)
	}
}

// 若 loop 正在运行且从loop goruntine 之外添加 task，应当设置 coroSafe=false
// target 为启动的协程
// args 为参数
// fut 为完成后设置结果的目标
func (loop *EventLoop[ArgsT, RetT]) CreateAndAddTask(coroSafe bool, target Step[ArgsT, RetT], args ArgsT, fut *Future[ArgsT, RetT], shim func(RetT) ArgsT) {
	event := NewReadyTask(target, args)
	// 将当前协程挂起并在新协程有结果时拉起当前协程
	event.Aux.resultShim = shim
	event.Aux.resultFut = fut
	loop.addTask(coroSafe, event)
}

// 运行当前所有就绪协程
func (loop *EventLoop[ArgsT, RetT]) runReadyTasks() {
	// 取出当前就绪的任务
	thisTasks := loop.events
	loop.events = make([]Event[ArgsT, RetT], 0)
	// 运行完成当前就绪的任务
	for _, task := range thisTasks {
		aux := task.Aux
		if !aux.yieldAlreadyReceivedValue() {
			panic("a yield currently not received any ret")
		}
		handle := loop.createHandle(task.Aux)
		nextStep := task.Next
		data := task.Data
		loop.runningCoroIdx = aux.uniqueIdx
		yieldReason := nextStep(handle, data)
		loop.runningCoroIdx = 0
		switch reason := yieldReason.(type) {
		default:
			panic("unknown yield reason ")
		case YieldByFinish[ArgsT, RetT]:
			// 未完成的事件数量 -=1
			loop.unfinishTasks -= 1
			if aux.resultFut != nil {
				aux.resultFut.SetResult(aux.resultShim(reason.Result))
				aux.resultFut = nil
				aux.resultShim = nil
			}
			continue
		case YieldByOuterResume[ArgsT, RetT]:
			continue
		case YieldBySleep[ArgsT, RetT]:
			// 是休眠，由 scheduler 管理
			wakeTime := reason.AwakeAt
			nextStep := reason.next
			if (wakeTime).Before(time.Now().Add(time.Nanosecond * -10)) {
				// 如果是 sleep(0) 立刻重新放回队列,这里减去一个极小值，从而认为在这个时间之内的时间也应该现在被调度
				var emptyArgs ArgsT
				handle.resumeNonCoroSafe(nextStep, emptyArgs)
			} else {

				loop.sleepingTasks.push(sleepingTask[ArgsT, RetT]{Next: nextStep, Resume: handle.resumeNonCoroSafe, AwakeAt: wakeTime})
			}
		case YieldByAwait[ArgsT, RetT]:
			// 本质上，这等于新创建了一个协程并放入调度队列中，同时把当前任务挂起，并要求新协程执行完后恢复任务
			// 将当前协程挂起并在新协程有结果时拉起当前协程
			fut := &Future[ArgsT, RetT]{}
			fut.AddWaiter(handle.resumeNonCoroSafe, reason.next)
			loop.CreateAndAddTask(true, reason.Target, reason.Args, fut, reason.Shim)
		}
	}
}

func (loop *EventLoop[ArgsT, RetT]) wakeDueSleepingTasks() {
	now := time.Now()
	for !loop.sleepingTasks.isEmpty() {
		if loop.sleepingTasks.first().AwakeAt.After(now) {
			return
		}
		task := loop.sleepingTasks.pop()
		var emptyArgs ArgsT
		task.Resume(task.Next, emptyArgs)
	}
}

// step 只负责推进计算，不主动等待。
// events 由外部推入，返回值含义：
func (loop *EventLoop[ArgsT, RetT]) step(events []Event[ArgsT, RetT]) (nextWakeup time.Time, unfinishCount int) {
	loop.events = append(loop.events, events...)

	for {
		loop.wakeDueSleepingTasks()
		if len(loop.events) == 0 {
			if !loop.sleepingTasks.isEmpty() {
				next := loop.sleepingTasks.first().AwakeAt
				return next, loop.unfinishTasks
			}
			// 没有 ready/sleeping，告知外部尚未完成的任务数量
			// 此时 nextWakeup.IsZero()=true
			return time.Time{}, loop.unfinishTasks
		}
		loop.runReadyTasks()
	}
}
