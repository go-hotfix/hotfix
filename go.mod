module github.com/go-hotfix/hotfix

go 1.21

//replace github.com/go-hotfix/assembly => ../assembly

require (
	github.com/brahma-adshonor/gohook v1.1.9
	github.com/go-delve/delve v1.23.1
	github.com/go-hotfix/assembly v0.0.0-20241127040136-3936dfdaf772
)

require (
	github.com/cilium/ebpf v0.11.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/arch v0.6.0 // indirect
	golang.org/x/exp v0.0.0-20230224173230-c95f2b4c22f2 // indirect
	golang.org/x/sys v0.17.0 // indirect
)
