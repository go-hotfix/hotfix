//go:build go1.21 && !go1.22

package hotfix

import _ "unsafe"

//go:linkname _stopTheWorld runtime.stopTheWorld
func _stopTheWorld(reason uint8)

//go:linkname startTheWorld runtime.startTheWorld
func startTheWorld()

//go:nosplit
func stopTheWorld() {
	// stwUnknown stwReason = iota // "unknown"
	_stopTheWorld(0)
}
