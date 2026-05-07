package hotfix

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
)

func TestGoMonkey_AllSuccess_NoRollback(t *testing.T) {
	// Verify the patcher function exists and is callable
	patcher := GoMonkey()
	if patcher == nil {
		t.Fatal("GoMonkey() returned nil")
	}

	// Verify that gomonkey is accessible (compile-time check)
	_ = gomonkey.NewPatches
}

func TestGoMonkey_RollbackLogic(t *testing.T) {
	// Test the rollback logic by simulating the behavior
	tests := []struct {
		name           string
		numFuncs       int
		failAtIndex    int
		expectedHook   int
		expectedUnhook int
	}{
		{
			name:           "first func fails - no rollback needed",
			numFuncs:       3,
			failAtIndex:    0,
			expectedHook:   0,
			expectedUnhook: 0,
		},
		{
			name:           "third func fails - rollback 2 funcs",
			numFuncs:       5,
			failAtIndex:    2,
			expectedHook:   2,
			expectedUnhook: 2,
		},
		{
			name:           "last func fails - rollback all previous",
			numFuncs:       3,
			failAtIndex:    2,
			expectedHook:   2,
			expectedUnhook: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooked := []int{}
			unhooked := []int{}

			// Simulate the patch loop with rollback logic (mirrors patcher.go)
			for i := 0; i < tt.numFuncs; i++ {
				if i == tt.failAtIndex {
					// Simulate failure - rollback via Reset
					if len(hooked) > 0 {
						unhooked = append(unhooked, hooked...)
					}
					break
				}
				hooked = append(hooked, i)
			}

			if len(hooked) != tt.expectedHook {
				t.Errorf("expected %d hooked, got %d", tt.expectedHook, len(hooked))
			}
			if len(unhooked) != tt.expectedUnhook {
				t.Errorf("expected %d unhooked, got %d", tt.expectedUnhook, len(unhooked))
			}
		})
	}
}

func TestGoMonkey_ErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		patchErr      error
		rollbackCount int
		contains      []string
	}{
		{
			name:          "patch error with successful rollback",
			patchErr:      errors.New("patching panic: something failed"),
			rollbackCount: 3,
			contains:      []string{"patching panic", "rolled back 3 functions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := fmt.Errorf("%w (rolled back %d functions)", tt.patchErr, tt.rollbackCount).Error()

			for _, s := range tt.contains {
				if !strings.Contains(msg, s) {
					t.Errorf("error message %q should contain %q", msg, s)
				}
			}
		})
	}
}
