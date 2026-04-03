package async

import "time"

// Step 描述里一个可挂起的模型的一步，也即 async 函数的一步
// 若干 Step 共同构成一个 Generator/Async 函数，
// Step 返回让出控制权的原因，而框架根据让出控制权的原因执行进一步操作
type Step[ArgsT any, RetT any] func(handle Handle[ArgsT, RetT], args ArgsT) YieldReason[ArgsT, RetT]

type YieldReason[ArgsT any, RetT any] interface {
	YieldReasonObject(ArgsT, RetT)
}

// 一次性的恢复函数，当同一个 Resume 被多次调用时会 panic
// 这个是 corosafe 的，也就是说可以在 loop 运行时在 loop 之外被调用而没有线程安全问题
type Resume[ArgsT any, RetT any] func(next Step[ArgsT, RetT], args ArgsT)

// Handle 同时提供一个 一次性的恢复函数(实际上是执行下一个 Step)
// 和一些控制和辅助函数方便获得自身的标识和创建新的协程
type Handle[ArgsT any, RetT any] interface {
	// 一次性的恢复函数，当同一个 Resume 被多次调用时会 panic
	// 这个总是被 Loop 外函数使用，所以一定是 CoroSafe 的
	Resume(next Step[ArgsT, RetT], args ArgsT)
	CorotineId() int32
	CreateTask(target Step[ArgsT, RetT], args ArgsT, shim func(RetT) ArgsT) *Future[ArgsT, RetT]
}

type CanCreateYieldReason[ArgsT any, RetT any] interface {
	// 函数应该以 return Reason.SuspendAndYield(Resume,NextStep), true
	// 的方式结束当前对执行权占用，注册自身并返回执行权给调度器
	SuspendAndYield(resumeCoroSafe Resume[ArgsT, RetT], next Step[ArgsT, RetT]) (reasonForLoop YieldReason[ArgsT, RetT])
}

// 代表这个协程让出控制权是因为需要休眠
type YieldBySleep[ArgsT any, RetT any] struct {
	// 当达到指定 Sleep 时间时，next 需要被再次运行
	next Step[ArgsT, RetT]
	// AwakeAt 指定其需要 Sleep 到的时间，可能小于等于现在的时间
	AwakeAt time.Time
}

func (y YieldBySleep[ArgsT, RetT]) YieldReasonObject(ArgsT, RetT) {}
func (y YieldBySleep[ArgsT, RetT]) SuspendAndYield(_ Resume[ArgsT, RetT], next Step[ArgsT, RetT]) (reasonForLoop YieldReason[ArgsT, RetT]) {
	y.next = next
	return y
}

// 这个协程让出控制权是因为需要Target(args)的结果
type YieldByAwait[ArgsT any, RetT any] struct {
	// 当Target执行完时，需要恢复的 next
	next Step[ArgsT, RetT]
	// 将以 Args 为参数，创建一个运行 Target 的新协程
	Target Step[ArgsT, RetT]
	Args   ArgsT
	// 如何将 Target 的结果作为 Next 的输入？这里需要一个转换函数
	Shim func(ret RetT) ArgsT
}

func (y YieldByAwait[ArgsT, RetT]) YieldReasonObject(ArgsT, RetT) {}
func (y YieldByAwait[ArgsT, RetT]) SuspendAndYield(_ Resume[ArgsT, RetT], next Step[ArgsT, RetT]) (reasonForLoop YieldReason[ArgsT, RetT]) {
	y.next = next
	return y
}

// 代表框架无需执行任何操作，下一步会通过 Resume 从其他地方恢复，而无需框架额外操作
type YieldByOuterResume[ArgsT any, RetT any] struct{}

func (YieldByOuterResume[ArgsT, RetT]) YieldReasonObject(ArgsT, RetT) {}

// 代表因为协程执行完毕，已经获得最终结果才结束
type YieldByFinish[ArgsT any, RetT any] struct {
	Result RetT
}

func (YieldByFinish[ArgsT, RetT]) YieldReasonObject(ArgsT, RetT) {}

// 这里总是 isCoroSafe=true
type FutureWaitor[ArgsT any, RetT any] struct {
	Resume[ArgsT, RetT]
	Step[ArgsT, RetT]
}

type Future[ArgsT any, RetT any] struct {
	done bool
	// 最终的值
	value ArgsT
	// 当接收到最终值时，需要被唤醒的对象
	waiters []FutureWaitor[ArgsT, RetT]
	// 特殊处理，因为很多时候只有一个waiter
	waitor0Exist bool
	waitor0      FutureWaitor[ArgsT, RetT]
}

func (f *Future[ArgsT, RetT]) SetResult(value ArgsT) {
	if f.done {
		panic("future already resolved")
	}
	f.done = true
	f.value = value
	if f.waitor0Exist {
		f.waitor0.Resume(f.waitor0.Step, value)
		f.waitor0Exist = false
		f.waitor0.Resume = nil
		f.waitor0.Step = nil
	}
	for _, waiter := range f.waiters {
		waiter.Resume(waiter.Step, value)
	}
	f.waiters = nil
}

func (f *Future[ArgsT, RetT]) AddWaiter(resume Resume[ArgsT, RetT], next Step[ArgsT, RetT]) {
	if f.done {
		resume(next, f.value)
		return
	}
	if !f.waitor0Exist {
		f.waitor0Exist = true
		f.waitor0 = FutureWaitor[ArgsT, RetT]{Resume: resume, Step: next}
	} else {
		f.waiters = append(f.waiters, FutureWaitor[ArgsT, RetT]{Resume: resume, Step: next})
	}
}

// 相当于等待 future 的结果，把自己挂入 waitor 并返回
// 由于通常情况下 future 可能在协程之外被设置结果，所以这里的是 resumeCoroSafe
// 这个等待不需要框架处理，所以直接返回 nil 即可
func (f *Future[ArgsT, RetT]) SuspendAndYield(resumeCoroSafe Resume[ArgsT, RetT], next Step[ArgsT, RetT]) (reasonForLoop YieldReason[ArgsT, RetT]) {
	f.AddWaiter(resumeCoroSafe, next)
	return YieldByOuterResume[ArgsT, RetT]{}
}
