package router

import (
	"net/http"
	"time"
)

func TimeHandler(w http.ResponseWriter, r *http.Request) {
	// 编译插件时修改此处模拟业务代码变更
	var nowTimeStr = time.Now().Format(time.DateTime)
	//var nowTimeStr = time.Now().Format(time.RFC1123)
	w.Write([]byte(nowTimeStr + "\n"))
}
