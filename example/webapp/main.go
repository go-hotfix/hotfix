package main

import (
	"fmt"
	"net/http"

	"github.com/go-hotfix/hotfix"
	"webapp/service"
)

func main() {
	http.HandleFunc("/order", service.OrderHandler)
	http.HandleFunc("/hotfix", HotfixHandler)

	fmt.Println("server listening on :8080")
	http.ListenAndServe(":8080", nil)
}

func HotfixHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("applying hotfix...")

	res := hotfix.Hotfix("webapp_v1.so", hotfix.Package("webapp/service"))

	fmt.Fprintf(w, "patch: %s, cost: %s\n", res.Patch, res.Cost)
	if res.Err != nil {
		fmt.Fprintf(w, "patch failed: %s\n", res.Err)
	}
	fmt.Fprintln(w, "methods:")
	for i, name := range res.Methods {
		fmt.Fprintf(w, "  %d: %s\n", i, name)
	}
	fmt.Fprintln(w, "logs:")
	fmt.Fprintln(w, res.Message)
}
