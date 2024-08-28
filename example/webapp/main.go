package main

import (
	"fmt"
	"net/http"

	"github.com/go-hotfix/hotfix"
	"webapp/router"
)

var HotfixVersion string

func main() {
	http.HandleFunc("/now", router.TimeHandler)
	http.HandleFunc("/hotfix", HotfixHandler)

	http.ListenAndServe(":8080", nil)
}

func HotfixHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Printf("apply patching...\n")

	res := hotfix.Hotfix("webapp_v1.so", hotfix.Package("webapp/router"))

	fmt.Fprintln(w, fmt.Sprintf("patch: %s, cost: %s ", res.Patch, res.Cost.String()))
	if nil != res.Err {
		fmt.Fprintln(w, "patch failed: ", res.Err)
	}
	fmt.Fprintln(w, "methods:")
	for i, name := range res.Methods {
		fmt.Fprintln(w, "\t", i, ": ", name)
	}
	fmt.Fprintln(w, "logs:")
	fmt.Fprintln(w, res.Message)
}
