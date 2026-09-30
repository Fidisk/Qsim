package symbolic

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type SymbolicValue struct {
	Expr Expression
}

type Expression struct {
	Op  *Ops
	Val *Value
	Var *Variable
}

type Value struct {
	Num complex64
}

type Variable struct {
	Name string
}

type OpFunc struct {
	Parse func(kids []string) string
	Do    func(args []complex128) complex128
}

type Ops struct {
	Fn   *OpFunc
	Args []Expression
}

func joinPlus(kids []string) string {
	return strings.Join(kids, " + ")
}

func joinMinus(kids []string) string {
	if len(kids) == 0 {
		return "NaN"
	}
	out := kids[0]
	for _, k := range kids[1:] {
		out += " - " + k
	}
	return out
}

func joinStar(kids []string) string {
	return strings.Join(kids, "*")
}

func joinSlash(kids []string) string {
	if len(kids) == 0 {
		return "NaN"
	}
	out := kids[0]
	for _, k := range kids[1:] {
		out += "/" + k
	}
	return out
}

func prefixMinus(kids []string) string {
	if len(kids) != 1 {
		return "NaN"
	}
	return "-" + kids[0]
}

func callConj(kids []string) string {
	if len(kids) != 1 {
		return "NaN"
	}
	return "conj(" + kids[0] + ")"
}

func sumNums(args []complex128) complex128 {
	var s complex128
	for _, a := range args {
		s += a
	}
	return s
}

func subNums(args []complex128) complex128 {
	if len(args) != 2 {
		return complex(math.NaN(), math.NaN())
	}
	return args[0] - args[1]
}

func prodNums(args []complex128) complex128 {
	p := complex(1, 0)
	for _, a := range args {
		p *= a
	}
	return p
}

func divNums(args []complex128) complex128 {
	if len(args) != 2 {
		return complex(math.NaN(), math.NaN())
	}
	return args[0] / args[1]
}

func negNums(args []complex128) complex128 {
	if len(args) != 1 {
		return complex(math.NaN(), math.NaN())
	}
	return -args[0]
}

func conjNums(args []complex128) complex128 {
	if len(args) != 1 {
		return complex(math.NaN(), math.NaN())
	}
	return complex(real(args[0]), -imag(args[0]))
}

var OpAdd = &OpFunc{Parse: joinPlus, Do: sumNums}
var OpSub = &OpFunc{Parse: joinMinus, Do: subNums}
var OpMul = &OpFunc{Parse: joinStar, Do: prodNums}
var OpDiv = &OpFunc{Parse: joinSlash, Do: divNums}
var OpNeg = &OpFunc{Parse: prefixMinus, Do: negNums}
var OpConj = &OpFunc{Parse: callConj, Do: conjNums}

var zeroValue = &Value{}

func (e Expression) resolve() Expression {
	if e.Op == nil && e.Val == nil && e.Var == nil {
		return Expression{Val: zeroValue}
	}
	return e
}

func Apply(fn *OpFunc, args []Expression) Expression {
	resolved := make([]Expression, len(args))
	for i, a := range args {
		resolved[i] = a.resolve()
	}
	allVal := true
	nums := make([]complex128, len(resolved))
	for i, a := range resolved {
		if a.Val == nil {
			allVal = false
			break
		}
		nums[i] = complex128(a.Val.Num)
	}
	if allVal {
		return Expression{Val: &Value{Num: complex64(fn.Do(nums))}}
	}
	flat := resolved
	if fn == OpAdd || fn == OpMul {
		flat = make([]Expression, 0, len(resolved))
		for _, a := range resolved {
			if a.Op != nil && a.Op.Fn == fn {
				flat = append(flat, a.Op.Args...)
				continue
			}
			flat = append(flat, a)
		}
		merged := make([]Expression, 0, len(flat))
		var acc complex128
		seenConst := false
		for _, a := range flat {
			if a.Val != nil {
				if !seenConst {
					seenConst = true
					acc = complex128(a.Val.Num)
				} else if fn == OpAdd {
					acc += complex128(a.Val.Num)
				} else {
					acc *= complex128(a.Val.Num)
				}
				continue
			}
			merged = append(merged, a)
		}
		if seenConst {
			flat = append([]Expression{{Val: &Value{Num: complex64(acc)}}}, merged...)
		} else {
			flat = merged
		}
	}
	return Expression{Op: &Ops{Fn: fn, Args: flat}}
}

func nanValue() Expression {
	return Expression{Val: &Value{Num: complex64(complex(math.NaN(), math.NaN()))}}
}

func New(re, im float32) SymbolicValue {
	return SymbolicValue{Expression{Val: &Value{Num: complex(re, im)}}}
}

func FromComplex64(c complex64) SymbolicValue {
	return SymbolicValue{Expression{Val: &Value{Num: c}}}
}

func FromComplex128(c complex128) SymbolicValue {
	return SymbolicValue{Expression{Val: &Value{Num: complex64(c)}}}
}

func Symbol(name string) SymbolicValue {
	return SymbolicValue{Expression{Var: &Variable{Name: name}}}
}

func Zero() SymbolicValue {
	return SymbolicValue{}
}

func One() SymbolicValue {
	return SymbolicValue{Expression{Val: &Value{Num: 1}}}
}

func (s SymbolicValue) closed() (complex64, bool) {
	e := s.Expr.resolve()
	if e.Val != nil {
		return e.Val.Num, true
	}
	return 0, false
}

func (s SymbolicValue) IsClosed() bool {
	_, ok := s.closed()
	return ok
}

func (s SymbolicValue) ToComplex64() complex64 {
	c, ok := s.closed()
	if !ok {
		return complex64(complex(math.NaN(), math.NaN()))
	}
	return c
}

func (s SymbolicValue) ToComplex128() complex128 {
	c, ok := s.closed()
	if !ok {
		return complex(math.NaN(), math.NaN())
	}
	return complex128(c)
}

func (s SymbolicValue) Real() float32 {
	c, ok := s.closed()
	if !ok {
		return float32(math.NaN())
	}
	return real(c)
}

func (s SymbolicValue) Imag() float32 {
	c, ok := s.closed()
	if !ok {
		return float32(math.NaN())
	}
	return imag(c)
}

func (s SymbolicValue) Add(o SymbolicValue) SymbolicValue {
	return SymbolicValue{Apply(OpAdd, []Expression{s.Expr, o.Expr})}
}

func (s SymbolicValue) Neg() SymbolicValue {
	return SymbolicValue{Apply(OpNeg, []Expression{s.Expr})}
}

func (s SymbolicValue) Sub(o SymbolicValue) SymbolicValue {
	return SymbolicValue{Apply(OpSub, []Expression{s.Expr, o.Expr})}
}

func (s SymbolicValue) Mul(o SymbolicValue) SymbolicValue {
	return SymbolicValue{Apply(OpMul, []Expression{s.Expr, o.Expr})}
}

func (s SymbolicValue) Div(o SymbolicValue) SymbolicValue {
	return SymbolicValue{Apply(OpDiv, []Expression{s.Expr, o.Expr})}
}

func (s SymbolicValue) Conj() SymbolicValue {
	return SymbolicValue{Apply(OpConj, []Expression{s.Expr})}
}

func (s SymbolicValue) IsZero() bool {
	c, ok := s.closed()
	return ok && c == 0
}

func (s SymbolicValue) IsOne() bool {
	c, ok := s.closed()
	return ok && c == 1
}

func (s SymbolicValue) Equal(o SymbolicValue) bool {
	return equalExpr(s.Expr.resolve(), o.Expr.resolve())
}

func equalExpr(a, b Expression) bool {
	if a.Val != nil || b.Val != nil {
		return a.Val != nil && b.Val != nil && a.Val.Num == b.Val.Num
	}
	if a.Var != nil || b.Var != nil {
		return a.Var != nil && b.Var != nil && a.Var.Name == b.Var.Name
	}
	if a.Op == nil || b.Op == nil {
		return a.Op == nil && b.Op == nil
	}
	if a.Op.Fn != b.Op.Fn || len(a.Op.Args) != len(b.Op.Args) {
		return false
	}
	for i := range a.Op.Args {
		if !equalExpr(a.Op.Args[i], b.Op.Args[i]) {
			return false
		}
	}
	return true
}

func (s SymbolicValue) Substitute(name string, v SymbolicValue) SymbolicValue {
	return SymbolicValue{substExpr(s.Expr.resolve(), name, v.Expr.resolve())}
}

func substExpr(e Expression, name string, v Expression) Expression {
	if e.Var != nil {
		if e.Var.Name == name {
			return v
		}
		return e
	}
	if e.Val != nil {
		return e
	}
	mapped := make([]Expression, len(e.Op.Args))
	for i, k := range e.Op.Args {
		mapped[i] = substExpr(k, name, v)
	}
	return Apply(e.Op.Fn, mapped)
}

func (s SymbolicValue) Eval(env map[string]complex128) (complex128, error) {
	return evalExpr(s.Expr.resolve(), env)
}

func evalExpr(e Expression, env map[string]complex128) (complex128, error) {
	if e.Val != nil {
		return complex128(e.Val.Num), nil
	}
	if e.Var != nil {
		v, ok := env[e.Var.Name]
		if !ok {
			return 0, fmt.Errorf("unbound symbol %q", e.Var.Name)
		}
		return v, nil
	}
	nums := make([]complex128, len(e.Op.Args))
	for i, k := range e.Op.Args {
		v, err := evalExpr(k, env)
		if err != nil {
			return 0, err
		}
		nums[i] = v
	}
	return e.Op.Fn.Do(nums), nil
}

func (s SymbolicValue) String() string {
	e := s.Expr.resolve()
	if e.Val != nil {
		v := complex128(e.Val.Num)
		return fmt.Sprintf("%.2f%+.2fi", real(v), imag(v))
	}
	return renderExpr(e)
}

func renderExpr(e Expression) string {
	e = e.resolve()
	if e.Val != nil {
		return renderNum(complex128(e.Val.Num))
	}
	if e.Var != nil {
		return e.Var.Name
	}
	kids := make([]string, len(e.Op.Args))
	for i, k := range e.Op.Args {
		rk := k.resolve()
		s := renderExpr(rk)
		if rk.Op != nil && !isBareKid(rk) {
			s = "(" + s + ")"
		}
		kids[i] = s
	}
	return e.Op.Fn.Parse(kids)
}

func isBareKid(e Expression) bool {
	return e.Op != nil && (e.Op.Fn == OpNeg || e.Op.Fn == OpConj)
}

func renderNum(v complex128) string {
	r, im := real(v), imag(v)
	if im == 0 {
		return strconv.FormatFloat(r, 'g', -1, 32)
	}
	if r == 0 {
		return strconv.FormatFloat(im, 'g', -1, 32) + "i"
	}
	out := strconv.FormatFloat(r, 'g', -1, 32)
	if im < 0 {
		return "(" + out + strconv.FormatFloat(im, 'g', -1, 32) + "i)"
	}
	return "(" + out + "+" + strconv.FormatFloat(im, 'g', -1, 32) + "i)"
}
