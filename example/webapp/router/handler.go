package router

import (
	"net/http"
	"time"
)

func TimeHandler(w http.ResponseWriter, r *http.Request) {
	// 在编译脚本提示 `please modify v1 plugin ...` 时切换下列代码注释，模拟业务函数功能变更
	// Toggle the following code comments when the script prompts 'please modify v1 plugin ...' to simulate the function change
	//
	var nowTimeStr = time.Now().Format(time.DateTime)
	//var nowTimeStr = time.Now().Format(time.RFC1123)
	w.Write([]byte(nowTimeStr + "\n"))
}
