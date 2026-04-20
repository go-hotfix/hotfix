//go:build go1.23

package hotfix

import (
	"reflect"

	"github.com/go-hotfix/assembly"
	"github.com/go-hotfix/assembly/linkname"
)

// stwReason is an enumeration of reasons the world is stopping.
type stwReason uint8

// worldStop provides context from the stop-the-world required by the
// start-the-world.
type worldStop struct {
	reason           stwReason
	startedStopping  int64
	finishedStopping int64
	stoppingCPUTime  int64
}

var (
	_stopTheWorld  func(reason stwReason) worldStop
	_startTheWorld func(w worldStop)
	_stopFlag      worldStop
)

func init() {
	// Runtime function discovery via moduledata — bypasses go:linkname restriction in Go 1.23+
	stopTheWorldPC := linkname.FuncPCForName("runtime.stopTheWorld")
	if stopTheWorldPC == 0 {
		panic("go-hotfix: runtime function not found: runtime.stopTheWorld")
	}
	startTheWorldPC := linkname.FuncPCForName("runtime.startTheWorld")
	if startTheWorldPC == 0 {
		panic("go-hotfix: runtime function not found: runtime.startTheWorld")
	}

	_stopTheWorld = assembly.CreateFuncForCodePtr(
		reflect.TypeOf(_stopTheWorld), uint64(stopTheWorldPC),
	).Interface().(func(stwReason) worldStop)

	_startTheWorld = assembly.CreateFuncForCodePtr(
		reflect.TypeOf(_startTheWorld), uint64(startTheWorldPC),
	).Interface().(func(worldStop))
}

//go:nosplit
func startTheWorld() {
	_startTheWorld(_stopFlag)
	_stopFlag = worldStop{}
}

//go:nosplit
func stopTheWorld() {
	_stopFlag = _stopTheWorld(0)
}
