package hotfix

import (
	"bytes"
	"fmt"
	"log"
	"plugin"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-hotfix/assembly"
)

// exclusivity is a global lock that prevents concurrent hotfix operations.
var exclusivity int32

// Request contains all the information needed to apply a hot patch.
type Request struct {
	// Logger receives debug log output during the patching process.
	Logger *log.Logger
	// Patch is the file path of the loaded plugin (.so).
	Patch string
	// Methods is the list of fully qualified function names to patch.
	Methods []string
	// Assembly provides DWARF-based access to runtime type and function information.
	Assembly assembly.DwarfAssembly
	// OldFuncEntrys contains the original function entry points from the main binary.
	OldFuncEntrys []*proc.Function
	// OldFunctions holds callable reflect.Values pointing to the original function entry points.
	OldFunctions []reflect.Value
	// NewFunctions holds callable reflect.Values from the loaded plugin.
	NewFunctions []reflect.Value
}

// Result contains the outcome of a hotfix operation.
type Result struct {
	// Assembly is the DWARF assembly used during the operation.
	Assembly assembly.DwarfAssembly
	// Patch is the resolved file path of the loaded plugin.
	Patch string
	// Methods is the list of function names that were patched.
	Methods []string
	// Cost is the total wall-clock time of the operation.
	Cost time.Duration
	// Err is non-nil if the operation failed.
	Err error
	// Message contains the debug log output from the operation.
	Message string
}

// Hotfix applies a hot patch using the default gomonkey-based patcher.
// libPath is the file path of the plugin (.so) built from the fixed source.
// funcPicker selects which functions to patch.
func Hotfix(libPath string, funcPicker FuncPicker) Result {
	return DoHotfix(libPath, funcPicker, GoMonkey())
}

// DoHotfix applies a hot patch using a custom FuncPatcher implementation.
func DoHotfix(libPath string, funcPicker FuncPicker, funcPatcher FuncPatcher) (result Result) {
	var start = time.Now()
	var funcNames []string
	var returnErr error
	var output bytes.Buffer
	var logger = log.New(&output, "[hotfix] ", log.LstdFlags|log.Lshortfile)

	defer func() {
		rr := recover()

		result.Patch = libPath
		result.Methods = funcNames
		result.Cost = time.Since(start)
		result.Message = strings.TrimSpace(output.String())
		result.Err = returnErr
		if nil != rr {
			err, ok := rr.(error)
			if !ok {
				err = fmt.Errorf("%v", rr)
			}
			if returnErr == nil {
				result.Err = fmt.Errorf("%w\n%s", err, debug.Stack())
			} else {
				result.Err = fmt.Errorf("%s: %w\n%s", returnErr.Error(), err, debug.Stack())
			}
		}
	}()

	// Acquire exclusive lock — only one hotfix can run at a time.
	if !atomic.CompareAndSwapInt32(&exclusivity, 0, 1) {
		returnErr = fmt.Errorf("an other hotfix in processing")
		return
	}
	defer atomic.StoreInt32(&exclusivity, 0)

	logger.Printf("env: %s/%s go%s %dbit trampoline=%d", runtime.GOOS, runtime.GOARCH, runtime.Version()[2:], archMode, jumpCodeSize)

	t0 := time.Now()

	// Load DWARF debug info for the main binary.
	logger.Printf("loading main binary debug symbols")
	if result.Assembly, returnErr = assembly.NewDwarfAssembly(); nil != returnErr {
		returnErr = fmt.Errorf("main assembly load failed: %w", returnErr)
		return
	}

	logger.Printf("main binary loaded, cost: %s", time.Since(t0))

	for plug := range result.Assembly.Plugins() {
		logger.Printf("loaded library: %s", plug)
	}

	// Resolve the list of functions to patch.
	t1 := time.Now()
	logger.Printf("resolving target functions")
	funcNames, returnErr = funcPicker(result.Assembly)
	if nil != returnErr {
		return
	}

	if 0 == len(funcNames) {
		returnErr = fmt.Errorf("empty functions")
		return
	}

	funcNames = uniqStrings(funcNames)
	sort.Strings(funcNames)

	// Verify each function exists in the main binary and has enough code space.
	oldFuncEntrys := make([]*proc.Function, 0, len(funcNames))
	for _, name := range funcNames {
		entry, err := result.Assembly.FindFunc(name)
		if nil != err {
			returnErr = fmt.Errorf("%w: function not found: %s", err, name)
			return
		}

		logger.Printf("  %s: entry=%#x size=%d", name, entry.Entry, entry.End-entry.Entry)

		// The jump code must fit within the original function body.
		if size := entry.End - entry.Entry; size < jumpCodeSize {
			returnErr = fmt.Errorf("jump code overflow: %s, size: %d, required: %d", name, size, jumpCodeSize)
			return
		}

		oldFuncEntrys = append(oldFuncEntrys, entry)
	}

	logger.Printf("resolved %d target(s), cost: %s", len(funcNames), time.Since(t1))

	// Load the plugin into the process address space.
	t2 := time.Now()
	logger.Printf("loading plugin: %s", libPath)
	if _, err := plugin.Open(libPath); nil != err {
		returnErr = err
		return
	}

	// Resolve the plugin's base address in the main process.
	lib, addr, err := result.Assembly.FindPlugin(libPath)
	if nil != err {
		returnErr = fmt.Errorf("%w: plugin not found: %s", err, libPath)
		return
	}

	if "" == lib {
		returnErr = fmt.Errorf("search plugin image failed: %s", libPath)
		return
	}

	logger.Printf("plugin loaded: %s, cost: %s", lib, time.Since(t2))

	libPath = lib

	// Load DWARF debug info for the plugin.
	t3 := time.Now()
	logger.Printf("loading plugin debug symbols")
	if err = result.Assembly.LoadImage(lib, addr); nil != err {
		returnErr = fmt.Errorf("%w: load plugin assembly failed: %s", err, lib)
		return
	}

	logger.Printf("plugin debug symbols loaded, cost: %s", time.Since(t3))

	// Verify that each target function exists in the plugin with a new entry point.
	t4 := time.Now()
	logger.Printf("validating %d function(s)", len(funcNames))

	newFunctions := make([]reflect.Value, 0, len(funcNames))
	oldFunctions := make([]reflect.Value, 0, len(funcNames))
	for i, name := range funcNames {
		hotfixFunc, err := result.Assembly.FindFuncValue(name, false)
		if nil != err {
			returnErr = fmt.Errorf("validating failed: %w: function not found: %s", err, name)
			return
		}

		// If the entry point is unchanged, the plugin doesn't contain this function.
		if newEntry := hotfixFunc.Pointer(); newEntry == uintptr(oldFuncEntrys[i].Entry) {
			returnErr = fmt.Errorf("validating failed: function not found in patch: %s", name)
			return
		}

		logger.Printf("  %s: %#x -> %#x", name, oldFuncEntrys[i].Entry, hotfixFunc.Pointer())

		newFunctions = append(newFunctions, hotfixFunc)

		// Create a callable wrapper for the old entry point using the new function's type.
		oldFunc := assembly.CreateFuncForCodePtr(hotfixFunc.Type(), oldFuncEntrys[i].Entry)
		oldFunctions = append(oldFunctions, oldFunc)
	}

	logger.Printf("validation complete, cost: %s", time.Since(t4))

	// Apply the binary patches.
	t5 := time.Now()
	logger.Printf("applying patch: %s (%d function(s))", lib, len(funcNames))
	returnErr = funcPatcher(Request{
		Logger:        logger,
		Patch:         libPath,
		Methods:       funcNames,
		Assembly:      result.Assembly,
		OldFuncEntrys: oldFuncEntrys,
		OldFunctions:  oldFunctions,
		NewFunctions:  newFunctions,
	})

	if nil != returnErr {
		logger.Printf("patch failed: %v, cost: %s", returnErr, time.Since(t5))
	} else {
		logger.Printf("patch applied successfully, cost: %s", time.Since(t5))
	}

	return
}

var archMode = 64

// jumpCodeSize is the size of the jump instruction generated by gomonkey:
// arm64: 24 bytes (4x MOVZ/MOVK + LDR + BR), amd64: 14 bytes.
var jumpCodeSize uint64

func init() {
	sz := unsafe.Sizeof(uintptr(0))
	if sz == 4 {
		archMode = 32
	}
	switch runtime.GOARCH {
	case "arm64":
		jumpCodeSize = 24
	default:
		jumpCodeSize = 14
	}
}

// uniqStrings returns a deduplicated copy of the input slice, preserving order.
func uniqStrings(collection []string) []string {
	result := make([]string, 0, len(collection))
	seen := make(map[string]struct{}, len(collection))

	for _, item := range collection {
		if _, ok := seen[item]; ok {
			continue
		}

		seen[item] = struct{}{}
		result = append(result, item)
	}

	return result
}
