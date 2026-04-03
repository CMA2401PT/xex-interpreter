package async

import (
	"time"
)

type LoopRunner[ArgsT any, RetT any] struct {
	*EventLoop[ArgsT, RetT]
	receivedEvents     chan Event[ArgsT, RetT]
	externEventReciver ExternEventReceiver[ArgsT, RetT]
}

type channelEventReceiver[ArgsT any, RetT any] struct {
	ch chan Event[ArgsT, RetT]
}

func (r channelEventReceiver[ArgsT, RetT]) ReceiveEvent(ev Event[ArgsT, RetT]) {
	r.ch <- ev
}

func NewEventLoopRunner[ArgsT any, RetT any]() *LoopRunner[ArgsT, RetT] {
	receivedEvents := make(chan Event[ArgsT, RetT], 64)
	eventReceiver := channelEventReceiver[ArgsT, RetT]{ch: receivedEvents}
	runner := &LoopRunner[ArgsT, RetT]{
		EventLoop:          NewEventLoop(eventReceiver),
		receivedEvents:     receivedEvents,
		externEventReciver: eventReceiver,
	}
	return runner
}

// 读取一条当前就绪的事件
// 如果没有就绪事件，那么返回 nil
func (loop *LoopRunner[ArgsT, RetT]) queryEvent() (ev Event[ArgsT, RetT], ok bool) {
	select {
	case ev := <-loop.receivedEvents:
		return ev, true
	default:
		return ev, false
	}
}

// 等待直到至少一个事件就绪或达到 limit
// 当 ok=true 时代表一个收到了一个事件
func (loop *LoopRunner[ArgsT, RetT]) waitEventUntil(timeoutTime time.Time) (ev Event[ArgsT, RetT], ok bool) {
	if timeoutTime.IsZero() {
		ev := <-loop.receivedEvents
		return ev, true
	}
	waitDuration := time.Until(timeoutTime)
	if waitDuration <= 0 {
		return ev, false
	}
	timer := time.NewTimer(waitDuration)
	defer timer.Stop()
	select {
	case ev := <-loop.receivedEvents:
		return ev, true
	case <-timer.C:
	}
	return ev, false
}

// 等待直到至少一个事件就绪或达到 limit
// 当满足任意一者时，尝试读取所有就绪的任务，如果没有就返回 nil
func (loop *LoopRunner[ArgsT, RetT]) waitEventUntilAndReceiveAll(timeoutTime time.Time) (events []Event[ArgsT, RetT]) {
	ev, ok := loop.waitEventUntil(timeoutTime)
	if ok {
		events = make([]Event[ArgsT, RetT], 0)
		events = append(events, ev)
		for {
			ev, ok := loop.queryEvent()
			if !ok {
				break
			}
			events = append(events, ev)
		}
	}
	return events
}

func (r *LoopRunner[ArgsT, RetT]) RunUntilComplete() {
	nextWakeup := time.Now()
	unFinishCount := 0
	for {
		events := r.waitEventUntilAndReceiveAll(nextWakeup)
		nextWakeup, unFinishCount = r.EventLoop.step(events)
		if unFinishCount == 0 {
			return
		}
	}
}
