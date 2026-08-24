package engine

const (
	OpEqual        = "="
	OpNotEqual     = "!="
	OpLess         = "<"
	OpGreater      = ">"
	OpLessEqual    = "<="
	OpGreaterEqual = ">="
)

type EqualOp struct{}

func (EqualOp) Op() string { return OpEqual }

func (EqualOp) Compare(got, want Value) bool {
	if got.IsNum && want.IsNum {
		return got.Number == want.Number
	}
	return got.Text == want.Text
}

type NotEqualOp struct{}

func (NotEqualOp) Op() string { return OpNotEqual }

func (NotEqualOp) Compare(got, want Value) bool {
	return !EqualOp{}.Compare(got, want)
}

type LessOp struct{}

func (LessOp) Op() string { return OpLess }

func (LessOp) Compare(got, want Value) bool {
	if !got.IsNum || !want.IsNum {
		return false
	}
	return got.Number < want.Number
}

type GreaterOp struct{}

func (GreaterOp) Op() string { return OpGreater }

func (GreaterOp) Compare(got, want Value) bool {
	if !got.IsNum || !want.IsNum {
		return false
	}
	return got.Number > want.Number
}

type LessEqualOp struct{}

func (LessEqualOp) Op() string { return OpLessEqual }

func (LessEqualOp) Compare(got, want Value) bool {
	if !got.IsNum || !want.IsNum {
		return false
	}
	return got.Number <= want.Number
}

type GreaterEqualOp struct{}

func (GreaterEqualOp) Op() string { return OpGreaterEqual }

func (GreaterEqualOp) Compare(got, want Value) bool {
	if !got.IsNum || !want.IsNum {
		return false
	}
	return got.Number >= want.Number
}
