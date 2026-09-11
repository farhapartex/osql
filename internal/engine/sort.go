package engine

import (
	"container/heap"
	"strings"
)

const SortCap = 10000

type SortableField interface {
	SortKey(r Row) Value
}

func lessValue(a, b Value) bool {
	if a.IsNum && b.IsNum {
		return a.Number < b.Number
	}
	return a.Text < b.Text
}

func sameValue(a, b Value) bool {
	if a.IsNum && b.IsNum {
		return a.Number == b.Number
	}
	return a.Text == b.Text
}

func orderBy(field SortableField, descending bool) func(a, b Row) bool {
	return func(a, b Row) bool {
		keyA, keyB := field.SortKey(a), field.SortKey(b)
		if !sameValue(keyA, keyB) {
			if descending {
				return lessValue(keyB, keyA)
			}
			return lessValue(keyA, keyB)
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	}
}

type rowHeap struct {
	rows  []Row
	worse func(a, b Row) bool
}

func (h rowHeap) Len() int           { return len(h.rows) }
func (h rowHeap) Less(i, j int) bool { return h.worse(h.rows[i], h.rows[j]) }
func (h rowHeap) Swap(i, j int)      { h.rows[i], h.rows[j] = h.rows[j], h.rows[i] }
func (h *rowHeap) Push(x any)        { h.rows = append(h.rows, x.(Row)) }

func (h *rowHeap) Pop() any {
	last := len(h.rows) - 1
	row := h.rows[last]
	h.rows = h.rows[:last]
	return row
}

type SortSink struct {
	better     func(a, b Row) bool
	keep       int
	kept       rowHeap
	overflowed bool
}

func NewSortSink(field SortableField, descending bool, keep int) *SortSink {
	if keep <= 0 {
		keep = SortCap
	}
	better := orderBy(field, descending)
	return &SortSink{
		better: better,
		keep:   keep,
		kept:   rowHeap{worse: func(a, b Row) bool { return better(b, a) }},
	}
}

func (s *SortSink) Push(r Row) error {
	if len(s.kept.rows) < s.keep {
		heap.Push(&s.kept, r)
		return nil
	}

	s.overflowed = true
	if s.better(r, s.kept.rows[0]) {
		s.kept.rows[0] = r
		heap.Fix(&s.kept, 0)
	}
	return nil
}

func (s *SortSink) Rows() []Row {
	ordered := make([]Row, len(s.kept.rows))
	for i := len(ordered) - 1; i >= 0; i-- {
		ordered[i] = heap.Pop(&s.kept).(Row)
	}
	return ordered
}

func (s *SortSink) Overflowed() bool {
	return s.overflowed
}
