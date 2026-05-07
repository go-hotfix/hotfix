package hotfix

import (
	"fmt"
	"reflect"

	"github.com/agiledragon/gomonkey/v2"
)

// FuncPatcher applies binary patches to redirect function calls.
// Implementations receive a fully populated Request and must return
// a non-nil error if patching fails.
type FuncPatcher func(req Request) error

// GoMonkey returns a FuncPatcher that uses gomonkey to rewrite function
// entry points. It performs type validation before entering stop-the-world
// (STW) to ensure thread-safe binary patching. If patching fails partway
// through, all previously applied patches are rolled back.
func GoMonkey() FuncPatcher {
	return func(req Request) (err error) {
		logger := req.Logger

		logger.Printf("  validating function types")
		if err := validateFuncTypes(req.OldFunctions, req.NewFunctions); err != nil {
			return err
		}

		// Binary patching is non-atomic: STW ensures no goroutine is
		// executing the code being overwritten.
		logger.Printf("  stop-the-world: pausing all goroutines")
		stopTheWorld()

		defer func() {
			startTheWorld()
			logger.Printf("  stop-the-world: resuming all goroutines")
		}()

		logger.Printf("  rewriting %d function entry/entries", len(req.OldFunctions))

		patches := gomonkey.NewPatches()
		patched := 0

		defer func() {
			if r := recover(); r != nil {
				patches.Reset()
				patchErr := fmt.Errorf("patching failed: %v", r)
				if patched > 0 {
					err = fmt.Errorf("%w (rolled back %d functions)", patchErr, patched)
				} else {
					err = patchErr
				}
			}
		}()

		for i := 0; i < len(req.OldFunctions); i++ {
			patches.ApplyFunc(req.OldFunctions[i].Interface(), req.NewFunctions[i].Interface())
			patched++
		}

		return nil
	}
}

// validateFuncTypes checks that each old/new function pair has matching types.
func validateFuncTypes(oldFuncs, newFuncs []reflect.Value) error {
	if len(oldFuncs) != len(newFuncs) {
		return fmt.Errorf("function count mismatch: old=%d, new=%d", len(oldFuncs), len(newFuncs))
	}
	for i := 0; i < len(oldFuncs); i++ {
		oldType := oldFuncs[i].Type()
		newType := newFuncs[i].Type()
		if oldType != newType {
			return fmt.Errorf("type mismatch at index %d: target type(%s) and double type(%s) are different", i, oldType, newType)
		}
	}
	return nil
}
