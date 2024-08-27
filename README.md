# hotfix
hotfix is a golang function hot-fix solution inspired by [go-hotfix](https://github.com/lsg2020/go-hotfix)

> 警告: 目前尚未经过严格测试，请勿用于生产环境

> 注意: 不支持Windows

> 注意: 为了确保函数都能被修补，因此需要关闭函数内联，会因此损失一些性能

# Features
* 使用[Delve](https://github.com/go-delve/delve)加载可执行文件并共享对象调试符号以查找与函数路径名相对应的函数地址
* 补丁包使用 [go plugin](https://pkg.go.dev/plugin) 
* 线程安全, 参考 `runtime.GC` 使用 `stw` 确保所有协程都进入安全点从而实现线程安全的补丁 
* 运行时修复支持: 导出函数/私有函数/成员方法

# Limits
* 受限于[go plugin](https://pkg.go.dev/plugin)仅支持Linux, FreeBSD, macOS，其他平台目前不支持
* 加载的补丁包无法卸载，如果热修复次数过多可能导致较大的内存占用
* 不支持对闭包进行修复，需要热修复的逻辑不要放在闭包中
* 不能修改已有数据结构和函数签名，否则可能将导致程序崩溃，应该仅用于bug修复
* 编译时请保留调试符号，并且禁用函数内联`-gcflags=all=-l`
* 编译错误`invalid reference to xxxx` 是因为 `go1.23`开始限制了`go:linkname`功能，必须添加编译参数关闭限制`-ldflags=-checklinkname=0`
* 补丁包的的编译环境必须和主程序一致，包括go编译器版本，编译参数，依赖等，否则加载补丁包将会失败
* 补丁包的`main`包下面的`init`会首先调用一次，请注意不要重复初始化
* 打补丁包时`main`包必须产生变化，否则可能出现 `plugin already loaded`错误，推荐使用 `-ldflags="-X main.HotfixVersion=v1.0.1"`指定版本号， 确保每次编译补丁包都会有变化
* **该方案处于实验性质，尚未经过严格验证**

# Example

参考这个[例子](./example/webapp)项目

### License

The repository released under version 2.0 of the Apache License.