package hotfix

import (
	"fmt"

	"github.com/brahma-adshonor/gohook"
)

// FuncPatcher To apply function hot patching.
type FuncPatcher func(req Request) error

// GoMonkey Hot patching implementation based on monkey-patching.
func GoMonkey() FuncPatcher {
	return func(req Request) error {
		// 代码热修复采用monkey-patch机制实现函数调用重定向（重写跳转指令）
		// 因为写跳转指令是非原子性的，因此在多线程环境无法保证安全的重写跳转指令
		// 需要一些方案确保能安全的重写跳转指令
		// 1. ptrace 使用外部程序模拟调试器行为（挂起程序，如果程序正在函数中则单步执行直到跳出函数调用范围）
		// 2. 程序内部保证（模拟类似safe-point机制）
		// 3. 参考runtime.GC使程序进入stw状态后重写跳转指令

		// 这里采用第三种方案，使程序进入stw装后进行补丁操作
		// 如果线程不安全则采用stw的方式确保补丁能安全执行,避免线程安全问题
		if !req.ThreadSafe {
			req.Logger.Printf("enter stw...")
			stopTheWorld()
			req.Logger.Printf("enter stw... finished")

			defer func() {
				req.Logger.Printf("leave stw...")
				startTheWorld()
				req.Logger.Printf("leave stw... finished")
			}()
		}

		req.Logger.Printf("monkey patching...")

		// Track successfully patched functions for rollback on failure
		patched := make([]int, 0, len(req.OldFunctions))

		for i := 0; i < len(req.OldFunctions); i++ {
			if err := gohook.HookByIndirectJmp(req.OldFunctions[i].Interface(), req.NewFunctions[i].Interface(), nil); nil != err {
				patchErr := fmt.Errorf("patching failed: index: %d, func: %s, reason: %w", i, req.OldFuncEntrys[i].Name, err)

				// Rollback: unhook all previously patched functions in reverse order
				var rollbackErrs []error
				for j := len(patched) - 1; j >= 0; j-- {
					idx := patched[j]
					if unhookErr := gohook.UnHook(req.OldFunctions[idx].Interface()); unhookErr != nil {
						req.Logger.Printf("rollback failed: index: %d, func: %s, reason: %v", idx, req.OldFuncEntrys[idx].Name, unhookErr)
						rollbackErrs = append(rollbackErrs, fmt.Errorf("rollback failed: index: %d, func: %s: %w", idx, req.OldFuncEntrys[idx].Name, unhookErr))
					} else {
						req.Logger.Printf("rollback success: index: %d, func: %s", idx, req.OldFuncEntrys[idx].Name)
					}
				}

				if len(rollbackErrs) > 0 {
					return fmt.Errorf("%w (rolled back %d/%d functions, %d rollback errors: %v)", patchErr, len(patched)-len(rollbackErrs), len(patched), len(rollbackErrs), rollbackErrs)
				}

				return fmt.Errorf("%w (rolled back %d functions)", patchErr, len(patched))
			}
			patched = append(patched, i)
		}

		req.Logger.Printf("monkey patching... finished")
		return nil
	}
}
