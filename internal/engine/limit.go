package engine

type LimitSink struct {
	inner  RowSink
	limit  int
	taken  int
	filled bool
}

func NewLimitSink(inner RowSink, limit int) *LimitSink {
	return &LimitSink{inner: inner, limit: limit}
}

func (s *LimitSink) Push(r Row) error {
	if s.taken >= s.limit {
		s.filled = true
		return ErrStopWalk
	}

	if err := s.inner.Push(r); err != nil {
		return err
	}
	s.taken++

	if s.taken >= s.limit {
		s.filled = true
		return ErrStopWalk
	}
	return nil
}

func (s *LimitSink) Filled() bool {
	return s.filled
}

func (s *LimitSink) Taken() int {
	return s.taken
}
