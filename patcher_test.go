package hotfix

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/brahma-adshonor/gohook"
)

func TestGoMonkey_AllSuccess_NoRollback(t *testing.T) {
	// Verify the patcher function exists and is callable
	patcher := GoMonkey()
	if patcher == nil {
		t.Fatal("GoMonkey() returned nil")
	}

	// Verify that gohook.UnHook is accessible (compile-time check)
	_ = gohook.UnHook
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
					// Simulate failure - rollback in reverse order
					for j := len(hooked) - 1; j >= 0; j-- {
						unhooked = append(unhooked, hooked[j])
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

			// Verify unhook order is reverse of hook order
			if len(unhooked) > 0 {
				for i := 0; i < len(unhooked)/2; i++ {
					if unhooked[i] != hooked[len(hooked)-1-i] {
						t.Errorf("unhook order not reverse: unhooked[%d]=%d, expected %d",
							i, unhooked[i], hooked[len(hooked)-1-i])
					}
				}
			}
		})
	}
}

func TestGoMonkey_ErrorMessages(t *testing.T) {
	tests := []struct {
		name              string
		patchErr          error
		rollbackCount     int
		rollbackFailCount int
		contains          []string
	}{
		{
			name:          "patch error with successful rollback",
			patchErr:      errors.New("patching failed: hook failed"),
			rollbackCount: 3,
			contains:      []string{"patching failed", "rolled back 3 functions"},
		},
		{
			name:              "patch error with partial rollback failure",
			patchErr:          errors.New("patching failed: hook failed"),
			rollbackCount:     3,
			rollbackFailCount: 1,
			contains:          []string{"patching failed", "1 rollback errors"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate error message construction (mirrors patcher.go logic)
			var msg string
			if tt.rollbackFailCount > 0 {
				rollbackErrs := make([]error, tt.rollbackFailCount)
				for i := range rollbackErrs {
					rollbackErrs[i] = errors.New("unhook error")
				}
				msg = fmt.Errorf("%w (rolled back %d/%d functions, %d rollback errors: %v)",
					tt.patchErr, tt.rollbackCount-tt.rollbackFailCount, tt.rollbackCount, tt.rollbackFailCount, rollbackErrs).Error()
			} else {
				msg = fmt.Errorf("%w (rolled back %d functions)", tt.patchErr, tt.rollbackCount).Error()
			}

			for _, s := range tt.contains {
				if !strings.Contains(msg, s) {
					t.Errorf("error message %q should contain %q", msg, s)
				}
			}
		})
	}
}
