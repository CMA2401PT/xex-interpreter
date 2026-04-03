package async

type Heap[E any] struct {
	heapData  []E
	compareFn func(a, b E) (aPriorThanB bool)
}

func newHeap[E any](compareFn func(a, b E) (aPriorThanB bool)) *Heap[E] {
	return &Heap[E]{heapData: make([]E, 0), compareFn: compareFn}
}

// push add one element to h
func (h *Heap[E]) push(v E) {
	h.heapData = append(h.heapData, v)
	idx := len(h.heapData) - 1
	for idx > 0 {
		parent := (idx - 1) / 2
		if !h.compareFn(h.heapData[idx], h.heapData[parent]) {
			break
		}
		h.heapData[idx], h.heapData[parent] = h.heapData[parent], h.heapData[idx]
		idx = parent
	}
}

func (h *Heap[E]) isEmpty() bool {
	return len(h.heapData) == 0
}

// first return the element with highest priority but not pop
// panic if empty
func (h *Heap[E]) first() E {
	if h.isEmpty() {
		panic("heap is empty")
	}
	return h.heapData[0]
}

// pop return the element with highest priority with pop
func (h *Heap[E]) pop() E {
	if h.isEmpty() {
		panic("heap is empty")
	}
	ret := h.heapData[0]
	last := len(h.heapData) - 1
	h.heapData[0] = h.heapData[last]
	h.heapData = h.heapData[:last]
	if h.isEmpty() {
		return ret
	}
	idx := 0
	for {
		left := idx*2 + 1
		right := idx*2 + 2
		best := idx

		if left < len(h.heapData) && h.compareFn(h.heapData[left], h.heapData[best]) {
			best = left
		}
		if right < len(h.heapData) && h.compareFn(h.heapData[right], h.heapData[best]) {
			best = right
		}
		if best == idx {
			break
		}
		h.heapData[idx], h.heapData[best] = h.heapData[best], h.heapData[idx]
		idx = best
	}
	return ret
}
