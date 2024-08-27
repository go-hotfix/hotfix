//go:build !go1.21

package hotfix

import _ "unsafe"

//go:linkname _stopTheWorld runtime.stopTheWorld
func _stopTheWorld(reason string)

//go:linkname startTheWorld runtime.startTheWorld
func startTheWorld()

//go:nosplit
func stopTheWorld() {
	_stopTheWorld("hot-patching")
}
