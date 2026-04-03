package async

import (
	"reflect"
	"testing"
	"time"
)

func asyncSleep(handle Handle[any, any], args any) YieldReason[any, any] {
	sleepFor := args.(time.Duration)
	begin := time.Now()
	sleep := YieldBySleep[any, any]{AwakeAt: begin.Add(sleepFor)}
	return sleep.SuspendAndYield(handle.Resume, func(handle Handle[any, any], args any) YieldReason[any, any] {
		return YieldByFinish[any, any]{time.Since(begin)}
	})

}

func TestYieldBySleepResumesAndResolvesFuture(t *testing.T) {
	runner := NewEventLoopRunner[any, any]()
	fut := &Future[any, any]{}
	sleepFor := 15 * time.Millisecond

	runner.EventLoop.CreateAndAddTask(true, asyncSleep, sleepFor, fut, func(a any) any { return a })
	runner.RunUntilComplete()

	if !fut.done {
		t.Fatal("future not resolved after sleep task completed")
	}
	elapsed, ok := fut.value.(time.Duration)
	if !ok {
		t.Fatalf("future value type = %T, want time.Duration", fut.value)
	}
	if elapsed < sleepFor/2 {
		t.Fatalf("sleep elapsed = %v, want at least %v", elapsed, sleepFor/2)
	}
}

func TestExternalResumeContinuesSuspendedTask(t *testing.T) {
	runner := NewEventLoopRunner[any, any]()
	fut := &Future[any, any]{}
	trace := make([]string, 0, 2)

	waitExternal := func(handle Handle[any, any], args any) YieldReason[any, any] {
		trace = append(trace, "start")
		next := func(handle Handle[any, any], args any) YieldReason[any, any] {
			trace = append(trace, "resumed")
			return YieldByFinish[any, any]{args}
		}
		go func() {
			time.Sleep(10 * time.Millisecond)
			handle.Resume(next, "ready")
		}()
		return YieldByOuterResume[any, any]{}
	}

	runner.EventLoop.CreateAndAddTask(true, waitExternal, nil, fut, func(a any) any { return a })
	runner.RunUntilComplete()

	if !reflect.DeepEqual(trace, []string{"start", "resumed"}) {
		t.Fatalf("trace = %#v", trace)
	}
	if fut.value != "ready" {
		t.Fatalf("future value = %#v, want %q", fut.value, "ready")
	}
}

func TestYieldByAwaitResumesWithChildResult(t *testing.T) {
	runner := NewEventLoopRunner[any, any]()
	fut := &Future[any, any]{}
	trace := make([]string, 0, 4)

	child := func(handle Handle[any, any], args any) YieldReason[any, any] {
		trace = append(trace, "child:start")
		input := args.(int)
		sleep := YieldBySleep[any, any]{AwakeAt: time.Now().Add(10 * time.Millisecond)}
		return sleep.SuspendAndYield(handle.Resume, func(handle Handle[any, any], args any) YieldReason[any, any] {
			trace = append(trace, "child:resume")
			return YieldByFinish[any, any]{input + 1}
		})
	}

	awaitFn := func(handle Handle[any, any], args any) YieldReason[any, any] {
		trace = append(trace, "parent:start")
		await := YieldByAwait[any, any]{
			Target: child,
			Args:   41,
			Shim:   func(ret any) any { return ret },
		}

		return await.SuspendAndYield(handle.Resume, func(handle Handle[any, any], args any) YieldReason[any, any] {
			trace = append(trace, "parent:resume")
			return YieldByFinish[any, any]{args.(int) * 2}
		})
	}

	runner.EventLoop.CreateAndAddTask(true, awaitFn, nil, fut, func(a any) any { return a })
	runner.RunUntilComplete()

	if !reflect.DeepEqual(trace, []string{
		"parent:start",
		"child:start",
		"child:resume",
		"parent:resume",
	}) {
		t.Fatalf("trace = %#v", trace)
	}
	if fut.value != 84 {
		t.Fatalf("future value = %#v, want 84", fut.value)
	}
}

func TestCreateTaskReturnsFutureAndCoroutineIDsAreDistinct(t *testing.T) {
	runner := NewEventLoopRunner[any, any]()
	fut := &Future[any, any]{}
	trace := make([]string, 0, 4)
	var parentID, childID int32

	child := func(handle Handle[any, any], args any) YieldReason[any, any] {
		childID = handle.CorotineId()
		trace = append(trace, "child:run")
		return YieldByFinish[any, any]{7}
	}

	parent := func(handle Handle[any, any], args any) YieldReason[any, any] {
		parentID = handle.CorotineId()
		trace = append(trace, "parent:start")
		childFuture := handle.CreateTask(child, nil, func(a any) any { return a })
		trace = append(trace, "parent:after-create")
		return childFuture.SuspendAndYield(handle.Resume, func(resume Handle[any, any], args any) YieldReason[any, any] {
			trace = append(trace, "parent:resume")
			return YieldByFinish[any, any]{args}
		})
	}

	runner.EventLoop.CreateAndAddTask(true, parent, nil, fut, func(a any) any { return a })
	runner.RunUntilComplete()

	if parentID == 0 || childID == 0 {
		t.Fatalf("coroutine ids should be non-zero, got parent=%d child=%d", parentID, childID)
	}
	if parentID == childID {
		t.Fatalf("coroutine ids should be distinct, both were %d", parentID)
	}
	if !reflect.DeepEqual(trace, []string{
		"parent:start",
		"parent:after-create",
		"child:run",
		"parent:resume",
	}) {
		t.Fatalf("trace = %#v", trace)
	}
	if fut.value != 7 {
		t.Fatalf("future value = %#v, want 7", fut.value)
	}
}

func TestFutureAddWaiterRunsImmediatelyAfterResolution(t *testing.T) {
	fut := &Future[any, any]{}
	var got any
	called := 0

	fut.SetResult("done")
	fut.AddWaiter(func(next Step[any, any], args any) {
		called += 1
		got = args
		if next == nil {
			t.Fatal("next step should be forwarded to resume")
		}
	}, func(handle Handle[any, any], args any) YieldReason[any, any] {
		return YieldByFinish[any, any]{}
	})

	if called != 1 {
		t.Fatalf("resume called %d times, want 1", called)
	}
	if got != "done" {
		t.Fatalf("resume args = %#v, want %q", got, "done")
	}
}
