package attributes

type Side struct {
	SideCntPositive int32
	SideCntNegative int32
}

func NewSide(l, r int32) *Side {
	return &Side{
		SideCntPositive: l,
		SideCntNegative: r,
	}
}
