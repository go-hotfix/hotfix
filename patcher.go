package hotfix

import (
	"fmt"

	"github.com/brahma-adshonor/gohook"
)

type FuncPatcher func(req Request) error

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
		for i := 0; i < len(req.OldFunctions); i++ {
			if err := gohook.HookByIndirectJmp(req.OldFunctions[i].Interface(), req.NewFunctions[i].Interface(), nil); nil != err {
				return fmt.Errorf("patching failed: index: %d, func: %s, reason: %w", i, req.OldFuncEntrys[i].Name, err)
			}
		}
		req.Logger.Printf("monkey patching... finished")
		return nil
	}
}
