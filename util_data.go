package canvas

import (
	"fmt"
	"io"
	"math"
	"strings"
)

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type PriorityQueue[T Integer] struct {
	items []T
	keys  []int
	less  func(T, T) bool
}

func NewPriorityQueue[T Integer](less func(T, T) bool, capacity int) *PriorityQueue[T] {
	pq := &PriorityQueue[T]{}
	pq.Reset(less, capacity)
	return pq
}

func (q *PriorityQueue[T]) Reset(less func(T, T) bool, capacity int) {
	if capacity < cap(q.items) {
		q.items = make([]T, 0, capacity)
		q.keys = make([]int, 0, capacity)
	} else {
		q.items = q.items[:0]
		q.keys = q.keys[:0]
	}
	q.less = less
}

func (q *PriorityQueue[T]) Len() int {
	return len(q.items)
}

func (q *PriorityQueue[T]) Init() {
	n := len(q.items)
	for i := n/2 - 1; 0 <= i; i-- {
		q.down(i, n)
	}
}

func (q *PriorityQueue[T]) Append(t T) {
	q.items = append(q.items, t)
	if int(t) < len(q.keys) {
		q.keys[t] = len(q.items) - 1
	} else {
		for len(q.keys) < int(t) {
			q.keys = append(q.keys, -1)
		}
		q.keys = append(q.keys, len(q.items)-1)
	}
}

func (q *PriorityQueue[T]) Push(t T) {
	q.Append(t)
	q.up(len(q.items) - 1)
}

func (q *PriorityQueue[T]) Top() T {
	return q.items[0]
}

func (q *PriorityQueue[T]) Pop() T {
	n := len(q.items) - 1
	q.swap(0, n)
	q.down(0, n)

	item := q.items[n]
	q.items = q.items[:n]
	return item
}

func (q *PriorityQueue[T]) Find(t T) (int, bool) {
	if t < 0 || len(q.keys) <= int(t) {
		return -1, false
	}
	return q.keys[t], true
}

func (q *PriorityQueue[T]) Fix(i int) {
	if !q.down(i, len(q.items)) {
		q.up(i)
	}
}

func (q *PriorityQueue[T]) swap(i, j int) {
	q.items[i], q.items[j] = q.items[j], q.items[i]
	q.keys[q.items[i]] = i
	q.keys[q.items[j]] = j
}

// from container/heap
func (q *PriorityQueue[T]) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !q.less(q.items[j], q.items[i]) {
			break
		}
		q.swap(i, j)
		j = i
	}
}

func (q *PriorityQueue[T]) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if n <= j1 || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && q.less(q.items[j2], q.items[j1]) {
			j = j2 // = 2*i + 2  // right child
		}
		if !q.less(q.items[j], q.items[i]) {
			break
		}
		q.swap(i, j)
		i = j
	}
	return i0 < i
}

func (q *PriorityQueue[T]) Print(w io.Writer) {
	q2 := NewPriorityQueue[T](q.less, len(q.items))
	q2.items = append(q2.items, q.items...)
	q = q2

	n := len(q.items) - 1
	for 0 < n {
		q.swap(0, n)
		q.down(0, n)
		n--
	}
	width := int(math.Log10(math.Max(1.0, float64(len(q.items)-1)))) + 1
	for k := len(q.items) - 1; 0 <= k; k-- {
		fmt.Fprintf(w, "%*d %v\n", width, len(q.items)-1-k, q.items[k])
	}
	return
}

func (q *PriorityQueue[T]) String() string {
	sb := strings.Builder{}
	q.Print(&sb)
	str := sb.String()
	if 0 < len(str) {
		str = str[:len(str)-1]
	}
	return str
}
