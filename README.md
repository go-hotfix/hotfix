<div align="center" style="text-align: center">
<img src="logo.png" alt="go-hotfix" width="128px"/>
<h1>hotfix</h1>
<p>hotfix is a golang function hot-fix solution</p>
<hr/>
</div>


> 警告: 目前尚未经过严格测试，请勿用于生产环境  
> Warning: This has not been rigorously tested, do not use in production environment

> 注意: 不支持Windows  
> Note: Windows is not supported

> 注意: 为了确保函数都能被修补，需要关闭函数内联，会因此损失一些性能  
> Note: To ensure that all functions can be patched, function inlining needs to be disabled, which will result in a loss of some performance

## Features
* 支持指定包级别/类级别/函数级别热补丁支持
* Supported for hot patching at package/class/function
* 支持导出函数/私有函数/成员方法修补
* Support for exporting functions/private functions/member methods patching
* 基于[monkey-patch](github.com/brahma-adshonor/gohook) + [plugin](https://pkg.go.dev/plugin)机制实现
* Implemented based on [monkey-patch](github.com/brahma-adshonor/gohook) + [plugin](https://pkg.go.dev/plugin)
* 线程安全, 使用 `stw` 确保所有协程都进入安全点从而实现线程安全的补丁
* Thread safety, use `stw` to ensure that all coroutines enter safe points to hot patching

## Limits
* 受限于[go plugin](https://pkg.go.dev/plugin)仅支持Linux, FreeBSD, macOS，其他平台目前不支持
* The go plugin is currently only supported on Linux, FreeBSD, and macOS platforms; other platforms are not supported
* 加载的补丁包无法卸载，如果热修复次数过多可能导致较大的内存占用
* The loaded patch package cannot be uninstalled. Too many hot fixes may result in large memory usage
* 不支持对闭包进行修复，需要热修复的逻辑不要放在闭包中
* Closures are not supported, and logic that requires hotfixes should not be placed in closures
* 不能修改已有数据结构和函数签名，否则可能将导致程序崩溃，应该仅用于bug修复
* Cannot modify existing data structures and function signatures, as this may lead to program crashes. It should only be used for bug fixes
* 编译时请保留调试符号，并且禁用函数内联`-gcflags=all=-l`
* Please keep the debugging symbols when compiling, and disable function inline `-gcflags=all=-l`
* 编译错误`invalid reference to xxxx` 是因为 `go1.23`开始限制了`go:linkname`功能，必须添加编译参数关闭限制`-ldflags=-checklinkname=0`
* The compilation error `invalid reference to xxxx` is because `go1.23` began to limit the `go:linkname` function, and the compilation parameter must be added to turn off the restriction `-ldflags=-checklinkname=0`
* 补丁包的的编译环境必须和主程序一致，包括go编译器版本，编译参数，依赖等，否则加载补丁包将会失败
* The patch package's build environment must match that of the main program, including the Go compiler version, compilation parameters, dependencies, etc., otherwise loading the patch package will fail
* 补丁包的`main`包下面的`init`会首先调用一次，请注意不要重复初始化
* The 'init' under the 'main' package of the patch package will be called once first, be careful not to initialize it repeatedly
* 打补丁包时`main`包必须产生变化，否则可能出现 `plugin already loaded`错误，推荐使用 `-ldflags="-X main.HotfixVersion=v1.0.1"`指定版本号， 确保每次编译补丁包都会有变化
* The `main` package must be changed when applying the patch package, otherwise the `plugin already loaded` error may occur. It is recommended to use `-ldflags="-X main.HotfixVersion=v1.0.1"` to specify the version number to ensure that the patch package is compiled every time There will be changes
* **该方案处于实验性质，尚未经过严格验证**
* **The solution is experimental in nature and has not yet been rigorously validated**

## Example

参考这个[例子](./example/webapp)项目  
Refer to this [example](./example/webapp) project


## Acknowledgments
This project inspired by <a href="https://github.com/lsg2020/go-hotfix">lsg2020/go-hotfix</a>

## License

The repository released under version 2.0 of the Apache License.