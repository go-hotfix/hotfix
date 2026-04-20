package hotfix

import (
	"strings"
	"testing"
)

func TestFunc_ReturnsSpecifiedNames(t *testing.T) {
	names := []string{"example/data.TestAdd", "example/data.(*DataType).TestHotfix"}
	picker := Func(names...)

	result, err := picker(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(names) {
		t.Fatalf("expected %d names, got %d", len(names), len(result))
	}
	for i, name := range names {
		if result[i] != name {
			t.Errorf("result[%d] = %q, want %q", i, result[i], name)
		}
	}
}

func TestFunc_RejectsClosure(t *testing.T) {
	picker := Func("example/data.func1.2")
	_, err := picker(nil)
	if err == nil {
		t.Fatal("expected error for closure name")
	}
	if !strings.Contains(err.Error(), "closure unsupported") {
		t.Fatalf("expected closure error, got: %v", err)
	}
}

func TestFunc_Empty(t *testing.T) {
	picker := Func()
	result, err := picker(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %v", result)
	}
}

func TestAny_CombinesPickers(t *testing.T) {
	picker1 := Func("pkg.FuncA")
	picker2 := Func("pkg.FuncB", "pkg.FuncC")

	combined := Any(picker1, picker2)
	result, err := combined(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"pkg.FuncA", "pkg.FuncB", "pkg.FuncC"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d names, got %d: %v", len(expected), len(result), result)
	}
	for i, name := range expected {
		if result[i] != name {
			t.Errorf("result[%d] = %q, want %q", i, result[i], name)
		}
	}
}

func TestAny_PropagatesError(t *testing.T) {
	errPicker := Func("pkg.func1.2") // closure - will error
	combined := Any(errPicker)
	_, err := combined(nil)
	if err == nil {
		t.Fatal("expected error from failing picker")
	}
}

func TestClosureRegexp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isClosure bool
	}{
		{"normal function", "example/data.TestAdd", false},
		{"method", "example/data.(*DataType).Test", false},
		{"closure", "example/data.func1", true},
		{"nested closure", "example/data.func1.2", true},
		{"private function", "example/data.testPrivate", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := closureExp.MatchString(tt.input)
			if got != tt.isClosure {
				t.Errorf("closureExp.MatchString(%q) = %v, want %v", tt.input, got, tt.isClosure)
			}
		})
	}
}
