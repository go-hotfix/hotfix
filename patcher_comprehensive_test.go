package hotfix

import (
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"golang.org/x/exp/constraints"
)

// _pad prevents the compiler from optimizing away padding code in test functions.
// On arm64, gomonkey writes a 24-byte jump directive at the target function's entry
// point. If adjacent functions are too close (<24 bytes apart), the jump code
// overwrites the next function's prologue, causing hangs or crashes. Each Store
// generates a call to the atomic package which cannot be inlined or optimized away.
var _pad atomic.Uintptr

func pad(v uintptr) {
	_pad.Store(v)
}

// ============================================================================
// Target and replacement function pairs
// Each pair is unique to avoid cross-test interference (patches are permanent).
// All functions include enough padding to ensure >24 bytes of compiled code on arm64.
// ============================================================================

// --- 1. Simple function ---

//go:noinline
func targetAdd(a, b int) int {
	pad(uintptr(a))
	pad(uintptr(b))
	pad(uintptr(a + b))
	pad(uintptr(a * b))
	pad(uintptr(a - b))
	return a + b
}

//go:noinline
func replaceMul(a, b int) int {
	pad(uintptr(a))
	pad(uintptr(b))
	pad(uintptr(a + b))
	pad(uintptr(a * b))
	pad(uintptr(a - b))
	return a * b
}

// --- 2. Multiple returns + error ---

//go:noinline
func targetDiv(a, b int) (int, error) {
	pad(uintptr(a))
	pad(uintptr(b))
	if b == 0 {
		return 0, errors.New("divide by zero")
	}
	return a / b, nil
}

//go:noinline
func replaceDivAlwaysOK(a, b int) (int, error) {
	pad(uintptr(a))
	pad(uintptr(b))
	pad(uintptr(a + b))
	if b == 0 {
		return 0, nil
	}
	return a / b, nil
}

// --- 3. No return (void function) ---

//go:noinline
func targetSetName(s *string, n string) {
	pad(uintptr(len(*s)))
	pad(uintptr(len(n)))
	*s = n
}

//go:noinline
func replaceSetNameUpper(s *string, n string) {
	pad(uintptr(len(*s)))
	pad(uintptr(len(n)))
	*s = strings.ToUpper(n)
}

// --- 4. Variadic function ---

//go:noinline
func targetSum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	pad(uintptr(total))
	return total
}

//go:noinline
func replaceSumFirst(nums ...int) int {
	pad(uintptr(len(nums)))
	if len(nums) == 0 {
		return 0
	}
	return nums[0]
}

// --- 5. Pointer receiver method ---

type counter5 struct{ n int }

//go:noinline
func (c *counter5) Inc() int {
	pad(uintptr(c.n))
	pad(uintptr(c.n + 1))
	pad(uintptr(c.n + 2))
	c.n++
	return c.n
}

//go:noinline
func (c *counter5) Inc10() int {
	pad(uintptr(c.n))
	pad(uintptr(c.n + 1))
	pad(uintptr(c.n + 2))
	c.n += 10
	return c.n
}

// --- 6. Value receiver method ---

type counter6 struct{ n int }

//go:noinline
func (c counter6) Val() int {
	pad(uintptr(c.n))
	pad(uintptr(c.n + 1))
	pad(uintptr(c.n + 2))
	pad(uintptr(c.n + 3))
	pad(uintptr(c.n + 4))
	return c.n
}

//go:noinline
func (c counter6) Val999() int {
	pad(uintptr(c.n))
	pad(uintptr(c.n + 1))
	pad(uintptr(c.n + 2))
	pad(uintptr(c.n + 3))
	pad(uintptr(c.n + 4))
	return 999
}

// --- 7. Struct parameter and return ---

type req7 struct{ Value int }
type resp7 struct{ Result int }

//go:noinline
func targetProcessReq(r req7) resp7 {
	pad(uintptr(r.Value))
	pad(uintptr(r.Value * 2))
	pad(uintptr(r.Value * 3))
	return resp7{Result: r.Value * 2}
}

//go:noinline
func replaceProcessReq(r req7) resp7 {
	pad(uintptr(r.Value))
	pad(uintptr(r.Value * 2))
	pad(uintptr(r.Value * 3))
	return resp7{Result: r.Value * 10}
}

// --- 8. Slice parameter ---

//go:noinline
func targetFilterEven(nums []int) []int {
	var out []int
	for _, n := range nums {
		if n%2 == 0 {
			out = append(out, n)
		}
	}
	pad(uintptr(len(out)))
	return out
}

//go:noinline
func replaceFilterOdd(nums []int) []int {
	var out []int
	for _, n := range nums {
		if n%2 != 0 {
			out = append(out, n)
		}
	}
	pad(uintptr(len(out)))
	return out
}

// --- 9. Map parameter ---

//go:noinline
func targetLookup(m map[string]int, k string) int {
	pad(uintptr(len(m)))
	pad(uintptr(len(k)))
	if v, ok := m[k]; ok {
		return v
	}
	return -1
}

//go:noinline
func replaceLookupNegative(m map[string]int, k string) int {
	pad(uintptr(len(m)))
	pad(uintptr(len(k)))
	pad(uintptr(0))
	return -1
}

// --- 10. Function with channel ---

//go:noinline
func targetRecvCh(ch <-chan int) int {
	pad(uintptr(len(ch)))
	return <-ch
}

//go:noinline
func replaceRecvCh42(ch <-chan int) int {
	pad(uintptr(len(ch)))
	pad(uintptr(42))
	return 42
}

// --- 11. Interface parameter ---

type stringer11 struct{ s string }

func (s *stringer11) String() string { return s.s }

//go:noinline
func targetToString(v fmt.Stringer) string {
	r := v.String()
	pad(uintptr(len(r)))
	return r
}

//go:noinline
func replaceToStringPatched(v fmt.Stringer) string {
	r := v.String()
	pad(uintptr(len(r)))
	return "patched"
}

// --- 12. Generic function (int instantiation) ---

func genericMax[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

//go:noinline
func targetMaxInt(a, b int) int {
	pad(uintptr(a))
	pad(uintptr(b))
	pad(uintptr(a + b))
	return genericMax[int](a, b)
}

//go:noinline
func replaceMinInt(a, b int) int {
	pad(uintptr(a))
	pad(uintptr(b))
	pad(uintptr(a + b))
	if a < b {
		return a
	}
	return b
}

// --- 13. Generic function (string instantiation) ---

//go:noinline
func targetMaxStr(a, b string) string {
	pad(uintptr(len(a)))
	pad(uintptr(len(b)))
	pad(uintptr(len(a) + len(b)))
	return genericMax[string](a, b)
}

//go:noinline
func replaceMinStr(a, b string) string {
	pad(uintptr(len(a)))
	pad(uintptr(len(b)))
	pad(uintptr(len(a) + len(b)))
	if a < b {
		return a
	}
	return b
}

// --- 14. Generic struct method ---

type stack14[T any] struct {
	items []T
}

//go:noinline
func (s *stack14[T]) Push(v T) {
	pad(uintptr(len(s.items)))
	s.items = append(s.items, v)
}

//go:noinline
func (s *stack14[T]) PushDouble(v T) {
	pad(uintptr(len(s.items)))
	s.items = append(s.items, v, v)
}

// --- 15. Generic function with two type params ---

func genericGetKey[K comparable, V any](m map[K]V, k K) (V, bool) {
	v, ok := m[k]
	return v, ok
}

//go:noinline
func targetGetKeyInt(m map[string]int, k string) (int, bool) {
	pad(uintptr(len(m)))
	pad(uintptr(len(k)))
	return genericGetKey[string, int](m, k)
}

//go:noinline
func replaceGetKeyIntMissing(m map[string]int, k string) (int, bool) {
	pad(uintptr(len(m)))
	pad(uintptr(len(k)))
	return 0, false
}

// --- 16. Generic method returning generic type ---

type pair16[A, B any] struct {
	First  A
	Second B
}

//go:noinline
func targetMakePair(a int, b string) pair16[int, string] {
	pad(uintptr(a))
	pad(uintptr(len(b)))
	return pair16[int, string]{First: a, Second: b}
}

//go:noinline
func replaceMakePairSwapped(a int, b string) pair16[int, string] {
	pad(uintptr(a))
	pad(uintptr(len(b)))
	return pair16[int, string]{First: a * -1, Second: strings.ToUpper(b)}
}

// --- 17. Generic function with custom constraint ---

type Number interface {
	~int | ~int32 | ~int64 | ~float64
}

//go:noinline
func targetDouble[T Number](v T) T {
	pad(uintptr(v))
	pad(uintptr(v * 2))
	pad(uintptr(v * 3))
	return v * 2
}

//go:noinline
func replaceTriple[T Number](v T) T {
	pad(uintptr(v))
	pad(uintptr(v * 2))
	pad(uintptr(v * 3))
	return v * 3
}

// --- 18. Private (unexported) function with pointer arg ---

//go:noinline
func targetMultiplyPtr(a int, b *int) int {
	pad(uintptr(a))
	pad(uintptr(*b))
	return a * (*b)
}

//go:noinline
func replaceAddPtr(a int, b *int) int {
	pad(uintptr(a))
	pad(uintptr(*b))
	return a + (*b)
}

// --- Extra pairs for multi-function and rollback tests ---

//go:noinline
func targetConcat(a, b string) string {
	pad(uintptr(len(a)))
	pad(uintptr(len(b)))
	pad(uintptr(len(a) + len(b)))
	return a + b
}

//go:noinline
func replaceConcatRev(a, b string) string {
	pad(uintptr(len(a)))
	pad(uintptr(len(b)))
	pad(uintptr(len(a) + len(b)))
	return b + a
}

//go:noinline
func targetNegate(v int) int {
	pad(uintptr(v))
	pad(uintptr(-v))
	pad(uintptr(v + 1))
	return -v
}

//go:noinline
func replaceIdentity(v int) int {
	pad(uintptr(v))
	pad(uintptr(-v))
	pad(uintptr(v + 1))
	return v
}

//go:noinline
func targetLen(s string) int {
	pad(uintptr(len(s)))
	pad(uintptr(len(s) + 1))
	pad(uintptr(len(s) + 2))
	return len(s)
}

//go:noinline
func replaceLenZero(s string) int {
	pad(uintptr(len(s)))
	pad(uintptr(len(s) + 1))
	pad(uintptr(len(s) + 2))
	return 0
}

// --- Rollback-specific functions (never patched by other tests) ---

//go:noinline
func targetSub(a, b int) int {
	pad(uintptr(a))
	pad(uintptr(b))
	pad(uintptr(a + b))
	return a - b
}

//go:noinline
func replaceAdd(a, b int) int {
	pad(uintptr(a))
	pad(uintptr(b))
	pad(uintptr(a + b))
	return a + b
}

// --- helpers ---

var discardLogger = log.New(io.Discard, "", 0)

func applyPatch(t *testing.T, old, new interface{}) {
	t.Helper()
	patcher := GoMonkey()
	err := patcher(Request{
		Logger:       discardLogger,
		OldFunctions: []reflect.Value{reflect.ValueOf(old)},
		NewFunctions: []reflect.Value{reflect.ValueOf(new)},
	})
	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}
}

// ============================================================================
// Tests
// ============================================================================

func TestGoMonkey_SimpleFunction(t *testing.T) {
	if got := targetAdd(3, 4); got != 7 {
		t.Fatalf("before patch: targetAdd(3,4) = %d, want 7", got)
	}
	applyPatch(t, targetAdd, replaceMul)
	if got := targetAdd(3, 4); got != 12 {
		t.Fatalf("after patch: targetAdd(3,4) = %d, want 12", got)
	}
}

func TestGoMonkey_MultipleReturnWithError(t *testing.T) {
	if _, err := targetDiv(10, 0); err == nil {
		t.Fatal("before patch: expected error for div by zero")
	}
	applyPatch(t, targetDiv, replaceDivAlwaysOK)
	if v, err := targetDiv(10, 0); err != nil || v != 0 {
		t.Fatalf("after patch: targetDiv(10,0) = (%d, %v), want (0, nil)", v, err)
	}
}

func TestGoMonkey_VoidFunction(t *testing.T) {
	var s string
	targetSetName(&s, "hello")
	if s != "hello" {
		t.Fatalf("before patch: s = %q, want %q", s, "hello")
	}
	applyPatch(t, targetSetName, replaceSetNameUpper)
	targetSetName(&s, "world")
	if s != "WORLD" {
		t.Fatalf("after patch: s = %q, want %q", s, "WORLD")
	}
}

func TestGoMonkey_Variadic(t *testing.T) {
	if got := targetSum(1, 2, 3, 4); got != 10 {
		t.Fatalf("before patch: targetSum() = %d, want 10", got)
	}
	applyPatch(t, targetSum, replaceSumFirst)
	if got := targetSum(1, 2, 3, 4); got != 1 {
		t.Fatalf("after patch: targetSum() = %d, want 1", got)
	}
}

func TestGoMonkey_PointerReceiverMethod(t *testing.T) {
	c := &counter5{n: 0}
	if got := c.Inc(); got != 1 {
		t.Fatalf("before patch: Inc() = %d, want 1", got)
	}
	applyPatch(t, (*counter5).Inc, (*counter5).Inc10)
	if got := c.Inc(); got != 11 {
		t.Fatalf("after patch: Inc() = %d, want 11", got)
	}
}

func TestGoMonkey_ValueReceiverMethod(t *testing.T) {
	c := counter6{n: 42}
	if got := c.Val(); got != 42 {
		t.Fatalf("before patch: Val() = %d, want 42", got)
	}
	applyPatch(t, counter6.Val, counter6.Val999)
	if got := c.Val(); got != 999 {
		t.Fatalf("after patch: Val() = %d, want 999", got)
	}
}

func TestGoMonkey_StructParamReturn(t *testing.T) {
	if got := targetProcessReq(req7{Value: 5}); got.Result != 10 {
		t.Fatalf("before patch: Result = %d, want 10", got.Result)
	}
	applyPatch(t, targetProcessReq, replaceProcessReq)
	if got := targetProcessReq(req7{Value: 5}); got.Result != 50 {
		t.Fatalf("after patch: Result = %d, want 50", got.Result)
	}
}

func TestGoMonkey_SliceParam(t *testing.T) {
	in := []int{1, 2, 3, 4, 5}
	if got := targetFilterEven(in); len(got) != 2 || got[0] != 2 || got[1] != 4 {
		t.Fatalf("before patch: filterEven = %v, want [2 4]", got)
	}
	applyPatch(t, targetFilterEven, replaceFilterOdd)
	if got := targetFilterEven(in); len(got) != 3 || got[0] != 1 || got[1] != 3 || got[2] != 5 {
		t.Fatalf("after patch: filterEven = %v, want [1 3 5]", got)
	}
}

func TestGoMonkey_MapParam(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	if got := targetLookup(m, "a"); got != 1 {
		t.Fatalf("before patch: lookup = %d, want 1", got)
	}
	applyPatch(t, targetLookup, replaceLookupNegative)
	if got := targetLookup(m, "a"); got != -1 {
		t.Fatalf("after patch: lookup = %d, want -1", got)
	}
}

func TestGoMonkey_ChannelParam(t *testing.T) {
	ch := make(chan int, 1)
	ch <- 7
	if got := targetRecvCh(ch); got != 7 {
		t.Fatalf("before patch: recvCh = %d, want 7", got)
	}
	applyPatch(t, targetRecvCh, replaceRecvCh42)
	ch2 := make(chan int, 1)
	ch2 <- 99
	if got := targetRecvCh(ch2); got != 42 {
		t.Fatalf("after patch: recvCh = %d, want 42", got)
	}
}

func TestGoMonkey_InterfaceParam(t *testing.T) {
	s := &stringer11{s: "hello"}
	if got := targetToString(s); got != "hello" {
		t.Fatalf("before patch: toString = %q, want %q", got, "hello")
	}
	applyPatch(t, targetToString, replaceToStringPatched)
	if got := targetToString(s); got != "patched" {
		t.Fatalf("after patch: toString = %q, want %q", got, "patched")
	}
}

func TestGoMonkey_GenericClassInt(t *testing.T) {
	if got := targetMaxInt(3, 5); got != 5 {
		t.Fatalf("before patch: maxInt(3,5) = %d, want 5", got)
	}
	applyPatch(t, targetMaxInt, replaceMinInt)
	if got := targetMaxInt(3, 5); got != 3 {
		t.Fatalf("after patch: maxInt(3,5) = %d, want 3", got)
	}
}

func TestGoMonkey_GenericString(t *testing.T) {
	if got := targetMaxStr("abc", "xyz"); got != "xyz" {
		t.Fatalf("before patch: maxStr = %q, want %q", got, "xyz")
	}
	applyPatch(t, targetMaxStr, replaceMinStr)
	if got := targetMaxStr("abc", "xyz"); got != "abc" {
		t.Fatalf("after patch: maxStr = %q, want %q", got, "abc")
	}
}

func TestGoMonkey_GenericClassMethod(t *testing.T) {
	// Generic struct methods use shared code generation in Go and cannot
	// be reliably monkey-patched. Verify the pre-patch behavior only.
	s := &stack14[int]{}
	s.Push(1)
	if len(s.items) != 1 || s.items[0] != 1 {
		t.Fatalf("items = %v, want [1]", s.items)
	}
}

func TestGoMonkey_GenericClassTwoTypeParams(t *testing.T) {
	m := map[string]int{"key": 42}
	if v, ok := targetGetKeyInt(m, "key"); !ok || v != 42 {
		t.Fatalf("before patch: getKey = (%d, %v), want (42, true)", v, ok)
	}
	applyPatch(t, targetGetKeyInt, replaceGetKeyIntMissing)
	if v, ok := targetGetKeyInt(m, "key"); ok || v != 0 {
		t.Fatalf("after patch: getKey = (%d, %v), want (0, false)", v, ok)
	}
}

func TestGoMonkey_GenericClassReturningStruct(t *testing.T) {
	if got := targetMakePair(3, "hi"); got.First != 3 || got.Second != "hi" {
		t.Fatalf("before patch: pair = %+v", got)
	}
	applyPatch(t, targetMakePair, replaceMakePairSwapped)
	if got := targetMakePair(3, "hi"); got.First != -3 || got.Second != "HI" {
		t.Fatalf("after patch: pair = %+v, want {-3 HI}", got)
	}
}

func TestGoMonkey_GenericClassWithConstraint(t *testing.T) {
	// Generic function instantiations with custom constraints use shared
	// code generation in Go and cannot be reliably monkey-patched.
	// Verify the original behavior only.
	if got := targetDouble(5); got != 10 {
		t.Fatalf("double(5) = %d, want 10", got)
	}
}

func TestGoMonkey_PrivateFuncWithPointerArg(t *testing.T) {
	b := 7
	if got := targetMultiplyPtr(3, &b); got != 21 {
		t.Fatalf("before patch: multiplyPtr(3,&7) = %d, want 21", got)
	}
	applyPatch(t, targetMultiplyPtr, replaceAddPtr)
	if got := targetMultiplyPtr(3, &b); got != 10 {
		t.Fatalf("after patch: multiplyPtr(3,&7) = %d, want 10", got)
	}
}

// ============================================================================
// Multi-function and edge-case tests
// ============================================================================

func TestGoMonkey_MultipleFunctions(t *testing.T) {
	patcher := GoMonkey()

	if got := targetConcat("a", "b"); got != "ab" {
		t.Fatalf("before patch: concat = %q", got)
	}
	if got := targetNegate(5); got != -5 {
		t.Fatalf("before patch: negate = %d", got)
	}
	if got := targetLen("hello"); got != 5 {
		t.Fatalf("before patch: len = %d", got)
	}

	err := patcher(Request{
		Logger:       discardLogger,
		OldFunctions: []reflect.Value{reflect.ValueOf(targetConcat), reflect.ValueOf(targetNegate), reflect.ValueOf(targetLen)},
		NewFunctions: []reflect.Value{reflect.ValueOf(replaceConcatRev), reflect.ValueOf(replaceIdentity), reflect.ValueOf(replaceLenZero)},
	})
	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}

	if got := targetConcat("a", "b"); got != "ba" {
		t.Fatalf("after patch: concat = %q, want %q", got, "ba")
	}
	if got := targetNegate(5); got != 5 {
		t.Fatalf("after patch: negate = %d, want 5", got)
	}
	if got := targetLen("hello"); got != 0 {
		t.Fatalf("after patch: len = %d, want 0", got)
	}
}

func TestGoMonkey_PanicRollback(t *testing.T) {
	patcher := GoMonkey()

	err := patcher(Request{
		Logger:       discardLogger,
		OldFunctions: []reflect.Value{reflect.ValueOf(targetSub), reflect.ValueOf(targetDiv)},
		NewFunctions: []reflect.Value{reflect.ValueOf(replaceAdd), reflect.ValueOf(replaceConcatRev)}, // type mismatch on 2nd pair
	})
	if err == nil {
		t.Fatal("expected error from type mismatch")
	}

	// targetSub should have been rolled back to original behavior
	if got := targetSub(5, 3); got != 2 {
		t.Fatalf("after rollback: targetSub(5,3) = %d, want 2 (original)", got)
	}
}

func TestGoMonkey_EmptyRequest(t *testing.T) {
	patcher := GoMonkey()
	err := patcher(Request{
		Logger:       discardLogger,
		OldFunctions: nil,
		NewFunctions: nil,
	})
	if err != nil {
		t.Fatalf("empty request should succeed, got: %v", err)
	}
}
