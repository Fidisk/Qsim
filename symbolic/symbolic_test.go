package symbolic

import (
	"math"
	"strings"
	"testing"
)

func TestClosedMatchesComplex64(t *testing.T) {
	a := New(0.5, -0.25)
	b := New(2, 0)
	if got := a.Mul(b).ToComplex64(); got != a.ToComplex64()*b.ToComplex64() {
		t.Fatalf("Mul = %v", got)
	}
	if got := a.Add(b).ToComplex64(); got != a.ToComplex64()+b.ToComplex64() {
		t.Fatalf("Add = %v", got)
	}
	if got := a.Sub(b).ToComplex64(); got != a.ToComplex64()-b.ToComplex64() {
		t.Fatalf("Sub = %v", got)
	}
	if got := a.Div(b).ToComplex64(); got != a.ToComplex64()/b.ToComplex64() {
		t.Fatalf("Div = %v", got)
	}
	var c complex64 = complex(0.375, -1.25)
	if got := FromComplex64(c).ToComplex64(); got != c {
		t.Fatalf("round trip = %v", got)
	}
}

func TestTermCombination(t *testing.T) {
	a := Symbol("a")
	b := Symbol("b")
	got := a.Add(b).Mul(a.Sub(b))
	if s := got.String(); s != "(a + b)*(a - b)" {
		t.Fatalf("factored String = %q", s)
	}
	env := map[string]complex128{"a": 3, "b": 1}
	gv, err := got.Eval(env)
	if err != nil {
		t.Fatalf("Eval err %v", err)
	}
	want := a.Mul(a).Sub(b.Mul(b))
	wv, err := want.Eval(env)
	if err != nil {
		t.Fatalf("Eval err %v", err)
	}
	if gv != wv {
		t.Fatalf("got %v want %v", gv, wv)
	}
}

func TestSubstituteEval(t *testing.T) {
	e := New(0.5, 0).Mul(Symbol("c")).Add(New(2, 0))
	v, err := e.Eval(map[string]complex128{"c": 4})
	if err != nil || v != complex(4, 0) {
		t.Fatalf("Eval = %v %v", v, err)
	}
	if _, err := e.Eval(nil); err == nil || !strings.Contains(err.Error(), "c") {
		t.Fatalf("open Eval err = %v", err)
	}
}

func TestConjNesting(t *testing.T) {
	a := Symbol("a")
	if s := a.Conj().Conj().String(); s != "conj(conj(a))" {
		t.Fatalf("conj nesting = %q", s)
	}
	env := map[string]complex128{"a": complex(1, 2)}
	gv, err := a.Conj().Conj().Eval(env)
	if err != nil {
		t.Fatalf("Eval err %v", err)
	}
	wv, err := a.Eval(env)
	if err != nil {
		t.Fatalf("Eval err %v", err)
	}
	if gv != wv {
		t.Fatalf("got %v want %v", gv, wv)
	}
	if !New(1, 2).Conj().Equal(New(1, -2)) {
		t.Fatalf("closed conj")
	}
}

func TestDescriptorCoverage(t *testing.T) {
	type kase struct {
		text string
		fn   *OpFunc
		want complex128
	}
	cases := []kase{
		{"a+b", OpAdd, complex(3, 0)},
		{"a-b", OpSub, complex(-1, 0)},
		{"a*b", OpMul, complex(2, 0)},
		{"a/b", OpDiv, complex(0.5, 0)},
		{"-a", OpNeg, complex(-1, 0)},
		{"conj(a)", OpConj, complex(1, 0)},
	}
	for _, c := range cases {
		v, err := Parse(c.text)
		if err != nil {
			t.Fatalf("%s err %v", c.text, err)
		}
		if v.Expr.Op == nil || v.Expr.Op.Fn != c.fn {
			t.Fatalf("%s Fn = %v", c.text, v.Expr.Op)
		}
		if v.String() == "" {
			t.Fatalf("empty String: %s", c.text)
		}
		if _, err := v.Eval(nil); err == nil {
			t.Fatalf("open Eval should fail: %s", c.text)
		}
		w := v.Substitute("a", New(1, 0)).Substitute("b", New(2, 0))
		if !w.IsClosed() {
			t.Fatalf("substitute should close: %s -> %v", c.text, w)
		}
		got, err := w.Eval(nil)
		if err != nil || got != c.want {
			t.Fatalf("%s Eval = %v %v want %v", c.text, got, err, c.want)
		}
	}
	if got := OpAdd.Do(nil); got != 0 {
		t.Fatalf("add empty = %v", got)
	}
	if got := OpMul.Do(nil); got != 1 {
		t.Fatalf("mul empty = %v", got)
	}
	if got := OpDiv.Do(nil); !math.IsNaN(real(got)) {
		t.Fatalf("div empty = %v", got)
	}
	if got := OpDiv.Do([]complex128{1}); !math.IsNaN(real(got)) {
		t.Fatalf("div unary = %v", got)
	}
	if got := OpNeg.Do([]complex128{5}); got != -5 {
		t.Fatalf("neg = %v", got)
	}
	if got := OpConj.Do([]complex128{complex(1, 2)}); got != complex(1, -2) {
		t.Fatalf("conj = %v", got)
	}
}

func TestParseEntries(t *testing.T) {
	v, err := Parse("0.5c")
	if err != nil || !v.Equal(New(0.5, 0).Mul(Symbol("c"))) {
		t.Fatalf("0.5c = %v %v", v, err)
	}
	v, err = Parse("2a+3b")
	if err != nil {
		t.Fatalf("2a+3b err %v", err)
	}
	if got, err := v.Eval(map[string]complex128{"a": 1, "b": 2}); err != nil || got != complex(8, 0) {
		t.Fatalf("2a+3b Eval = %v %v", got, err)
	}
	v, err = Parse("a+2i")
	if err != nil {
		t.Fatalf("a+2i err %v", err)
	}
	if got, err := v.Eval(map[string]complex128{"a": 1}); err != nil || got != complex(1, 2) {
		t.Fatalf("a+2i Eval = %v %v", got, err)
	}
	v, err = Parse("i")
	if err != nil || !v.Equal(New(0, 1)) {
		t.Fatalf("i = %v %v", v, err)
	}
	v, err = Parse("1e-3+2i")
	if err != nil || !v.IsClosed() {
		t.Fatalf("1e-3+2i = %v %v", v, err)
	}
	v, err = Parse("1,2")
	if err != nil || !v.Equal(New(1, 2)) {
		t.Fatalf("1,2 = %v %v", v, err)
	}
	v, err = Parse("sqrt2")
	if err != nil || !v.IsClosed() || v.ToComplex64() != complex64(complex(math.Sqrt2, 0)) {
		t.Fatalf("sqrt2 = %v %v", v, err)
	}
	if _, err := Parse(""); err == nil {
		t.Fatalf("empty should fail")
	}
	if _, err := Parse("conj"); err == nil {
		t.Fatalf("bare conj should fail")
	}
}

func TestDivScalarStaysOpen(t *testing.T) {
	e := Symbol("a").Add(Symbol("b")).Div(New(2, 0))
	if e.IsClosed() {
		t.Fatalf("should stay open: %v", e)
	}
	v, err := e.Eval(map[string]complex128{"a": 1, "b": 3})
	if err != nil || v != complex(2, 0) {
		t.Fatalf("Eval = %v %v", v, err)
	}
}

func TestNormalizeOpenNoOp(t *testing.T) {
	amps := []SymbolicValue{Symbol("a"), One()}
	Normalize(amps)
	if !amps[0].Equal(Symbol("a")) || !amps[1].Equal(One()) {
		t.Fatalf("open normalize must not touch: %v", amps)
	}
}

func TestIsOne(t *testing.T) {
	if !One().IsOne() || Zero().IsOne() || Symbol("a").IsOne() {
		t.Fatalf("IsOne wrong")
	}
}

func TestParseParens(t *testing.T) {
	v, err := Parse("a(b+c)")
	if err != nil {
		t.Fatalf("a(b+c) err %v", err)
	}
	if got, err := v.Eval(map[string]complex128{"a": 2, "b": 3, "c": 4}); err != nil || got != complex(14, 0) {
		t.Fatalf("a(b+c) Eval = %v %v", v, err)
	}
	if s := v.String(); s != "a*(b + c)" {
		t.Fatalf("a(b+c) String = %q", s)
	}
	w, err := Parse("(a+b)(a-b)")
	if err != nil {
		t.Fatalf("(a+b)(a-b) err %v", err)
	}
	if s := w.String(); s != "(a + b)*(a - b)" {
		t.Fatalf("(a+b)(a-b) String = %q", s)
	}
	if got, err := w.Eval(map[string]complex128{"a": 3, "b": 1}); err != nil || got != complex(8, 0) {
		t.Fatalf("(a+b)(a-b) Eval = %v %v", got, err)
	}
	u, err := Parse("2(a+b)")
	if err != nil {
		t.Fatalf("2(a+b) err %v", err)
	}
	if got, err := u.Eval(map[string]complex128{"a": 1, "b": 2}); err != nil || got != complex(6, 0) {
		t.Fatalf("2(a+b) Eval = %v %v", got, err)
	}
	d, err := Parse("conj(a+1)")
	if err != nil {
		t.Fatalf("conj(a+1) err %v", err)
	}
	if s := d.String(); s != "conj((1 + a))" {
		t.Fatalf("conj(a+1) String = %q", s)
	}
	if got, err := d.Eval(map[string]complex128{"a": complex(1, 1)}); err != nil || got != complex(2, -1) {
		t.Fatalf("conj(a+1) Eval = %v %v", got, err)
	}
	q, err := Parse("a/b")
	if err != nil {
		t.Fatalf("a/b err %v", err)
	}
	if s := q.String(); s != "a/b" {
		t.Fatalf("a/b String = %q", s)
	}
	for _, bad := range []string{"(a+b", "a)", "()", "2*", "a/"} {
		if _, err := Parse(bad); err == nil {
			t.Fatalf("%q should fail", bad)
		}
	}
}
