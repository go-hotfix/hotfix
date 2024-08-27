//go:build go1.23

package hotfix

import _ "unsafe"

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

var _stopFlag worldStop

//go:linkname _stopTheWorld runtime.stopTheWorld
func _stopTheWorld(reason stwReason) worldStop

//go:linkname _startTheWorld runtime.startTheWorld
func _startTheWorld(w worldStop)

//go:nosplit
func startTheWorld() {
	_startTheWorld(_stopFlag)
	_stopFlag = worldStop{}
}

//go:nosplit
func stopTheWorld() {
	// stwUnknown stwReason = iota // "unknown"
	_stopFlag = _stopTheWorld(0)
}
