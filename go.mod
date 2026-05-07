module github.com/go-hotfix/hotfix

go 1.24

//replace github.com/go-hotfix/assembly => ../assembly

require (
	github.com/agiledragon/gomonkey/v2 v2.14.0
	github.com/go-delve/delve v1.26.3
	github.com/go-hotfix/assembly v0.0.0-20260507105919-4e689533b2ec
	golang.org/x/exp v0.0.0-20230224173230-c95f2b4c22f2
)

require (
	github.com/cilium/ebpf v0.11.0 // indirect
	golang.org/x/arch v0.11.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/telemetry v0.0.0-20241106142447-58a1122356f5 // indirect
)
