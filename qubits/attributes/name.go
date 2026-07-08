package attributes

type Name struct {
	Val string
}

func NewName(val string) *Name {
	return &Name{
		Val: val,
	}
}
