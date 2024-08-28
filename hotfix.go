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

var exclusivity int32

type Request struct {
	Logger        *log.Logger            // Debug logger.
	Patch         string                 // Plugin file.
	ThreadSafe    bool                   // Whether it is thread safe.
	Methods       []string               // Patching function list.
	Assembly      assembly.DwarfAssembly // Go runtime assembly.
	OldFuncEntrys []*proc.Function       // Old function entrys.
	OldFunctions  []reflect.Value        // Old function values.
	NewFunctions  []reflect.Value        // Plugin function values.
}

type Result struct {
	Assembly   assembly.DwarfAssembly
	Patch      string        // Plugin file
	ThreadSafe bool          // Whether it is thread safe, the default is false, use stw mechanism to ensure thread safety.
	Methods    []string      // Patching function list.
	Cost       time.Duration // Total of cost time.
	Err        error         // Patching failed reason.
	Message    string        // Patching debug message.
}

// Hotfix Apply hot patching by default.
func Hotfix(libPath string, funcPicker FuncPicker, threadSafe ...bool) Result {
	return DoHotfix(libPath, funcPicker, GoMonkey(), threadSafe...)
}

// DoHotfix Apply hot patching in a custom way.
func DoHotfix(libPath string, funcPicker FuncPicker, funcPatcher FuncPatcher, threadSafe ...bool) (result Result) {

	var start = time.Now()
	var funcNames []string
	var returnErr error
	var output bytes.Buffer
	var logger = log.New(&output, "[hotfix]", log.LstdFlags|log.Lshortfile)

	defer func() {
		rr := recover()

		result.Patch = libPath
		result.ThreadSafe = len(threadSafe) > 0 && threadSafe[0]
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

	// 获取全局独占锁
	// 不允许并发执行热修复
	if !atomic.CompareAndSwapInt32(&exclusivity, 0, 1) {
		returnErr = fmt.Errorf("an other hotfix in processing")
		return
	}

	// 释放全局独占锁
	defer atomic.StoreInt32(&exclusivity, 0)

	logger.Printf("arch: %s/%s, compiler: %s/%s, cpu: %d-bit, jump code size: %d", runtime.GOOS, runtime.GOARCH, runtime.Compiler, runtime.Version(), archMode, jumpCodeSize)

	t0 := time.Now()

	// 加载主程序集
	logger.Printf("loading main assembly ...")
	if result.Assembly, returnErr = assembly.NewDwarfAssembly(); nil != returnErr {
		returnErr = fmt.Errorf("main assembly load failed: %w", returnErr)
		return
	}

	t1 := time.Now()

	logger.Printf("load main assembly finished, cost: %s", t1.Sub(t0).String())

	// 输出当前已经加载的插件
	if plugins, addrs, err := result.Assembly.SearchPlugins(); nil == err {
		for i, plug := range plugins {
			if 0 != addrs[i] {
				logger.Printf("loaded dynamic library: %s@%#x", plug, addrs[i])
			}
		}
	}

	// 加载需要热修复的函数
	logger.Printf("lookup patch functions ...")
	funcNames, returnErr = funcPicker(result.Assembly)
	if nil != returnErr {
		return
	}

	if 0 == len(funcNames) {
		returnErr = fmt.Errorf("empty functions")
		return
	}

	// 删除重复的项目
	funcNames = uniqStrings(funcNames)
	// 排序一下
	sort.Strings(funcNames)

	// 检查待热更的函数是否存在
	oldFuncEntrys := make([]*proc.Function, 0, len(funcNames))
	for _, name := range funcNames {
		// 查找当前等待补丁的函数地址
		entry, err := result.Assembly.FindFuncEntry(name)
		if nil != err {
			returnErr = fmt.Errorf("%w: function not found: %s", err, name)
			return
		}

		logger.Printf("find function: %s, entry: %#x, codeSpace: %d", name, entry.Entry, entry.End-entry.Entry)

		// jump code 代码不能比原有的代码还长，否则将产生覆写，这里直接拒绝
		if size := entry.End - entry.Entry; size < jumpCodeSize {
			returnErr = fmt.Errorf("jump code overflow: %s, size: %d, required: %d", name, size, jumpCodeSize)
			return
		}

		oldFuncEntrys = append(oldFuncEntrys, entry)
	}

	t2 := time.Now()

	logger.Printf("lookup patch functions finished, cost: %s", t2.Sub(t1).String())

	// 加载动态库到进程空间
	logger.Printf("opening patch %s ...", libPath)
	if _, err := plugin.Open(libPath); nil != err {
		returnErr = err
		return
	}

	// 查找的插件在主进程中的地址
	lib, addr, err := result.Assembly.SearchPluginByName(libPath)
	if nil != err {
		returnErr = fmt.Errorf("%w: plugin not found: %s", err, libPath)
		return
	}

	// 插件查找失败
	if "" == lib {
		returnErr = fmt.Errorf("search plugin image failed: %s", libPath)
		return
	}

	t3 := time.Now()

	logger.Printf("opening patch %s finished, cost: %s", lib, t3.Sub(t2).String())

	// 使用完整路径
	libPath = lib

	// 加载插件符号表
	logger.Printf("load patch assembly ...")
	if err = result.Assembly.LoadImage(lib, addr); nil != err {
		returnErr = fmt.Errorf("%w: load plugin assembly failed: %s", err, lib)
		return
	}

	t4 := time.Now()
	logger.Printf("load patch assembly finished, cost: %s", t4.Sub(t3).String())

	logger.Printf("validating hotfix functions ... ")

	newFunctions := make([]reflect.Value, 0, len(funcNames))
	oldFunctions := make([]reflect.Value, 0, len(funcNames))
	for i, name := range funcNames {
		// 查找插件补丁类型
		hotfixFunc, err := result.Assembly.FindFunc(name, false)
		if nil != err {
			returnErr = fmt.Errorf("validating failed: %w: function not found: %s", err, name)
			return
		}

		// 如果补丁中存在某个函数则在LoadAssembly中会被替换为新函数对象，函数地址会变更为补丁函数地址
		// 如果指定的函数补丁中不存在那么无法对这个函数进行修补
		if newEntry := hotfixFunc.Pointer(); newEntry == uintptr(oldFuncEntrys[i].Entry) {
			returnErr = fmt.Errorf("validating failed: function not found in patch: %s", name)
			return
		}

		logger.Printf("validating hotfix function: %s, entry: %#x -> %#x", name, oldFuncEntrys[i].Entry, hotfixFunc.Pointer())

		newFunctions = append(newFunctions, hotfixFunc)

		// 统一旧函数对象类型(插件和主程序的类型不一样)
		oldFunc := assembly.CreateFuncForCodePtr(hotfixFunc.Type(), oldFuncEntrys[i].Entry)
		oldFunctions = append(oldFunctions, oldFunc)
	}

	t5 := time.Now()
	logger.Printf("validating hotfix functions ... finished, cost: %s", t5.Sub(t4).String())

	// 执行补丁操作
	logger.Printf("apply patch ... patch: %s, threadSafe: %v", lib, len(threadSafe) > 0 && threadSafe[0])
	returnErr = funcPatcher(Request{
		Logger:        logger,
		Patch:         libPath,
		ThreadSafe:    len(threadSafe) > 0 && threadSafe[0],
		Methods:       funcNames,
		Assembly:      result.Assembly,
		OldFuncEntrys: oldFuncEntrys,
		OldFunctions:  oldFunctions,
		NewFunctions:  newFunctions,
	})

	t6 := time.Now()

	if nil != returnErr {
		logger.Printf("apply patch failed: %v, cost: %s", returnErr, t6.Sub(t5).String())
	} else {
		logger.Printf("apply patch success, cost: %s", t6.Sub(t5).String())
	}

	return
}

var archMode = 64

func init() {
	sz := unsafe.Sizeof(uintptr(0))
	if sz == 4 {
		archMode = 32
	}
}

// jumpCodeSize count jump code size
var jumpCodeSize = uint64(len(genJumpCode(archMode, true, 0, 0)))

//go:linkname genJumpCode github.com/brahma-adshonor/gohook.genJumpCode
func genJumpCode(mode int, rdxIndirect bool, to, from uintptr) []byte

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
