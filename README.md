<div align="center">
<img src="logo.png" alt="go-hotfix" width="128px"/>
<h1>hotfix</h1>
<p>Runtime function hot-patching for Go applications</p>
</div>

---

> **Warning:** This project is experimental. Do not use in production without thorough testing.

## How It Works

`hotfix` patches running Go functions at runtime using [gomonkey](https://github.com/agiledragon/gomonkey) + [go plugin](https://pkg.go.dev/plugin):

1. Build a fixed version of your code as a `.so` plugin
2. Load the plugin into the running process
3. Redirect old function calls to the new implementations via binary patching

## Features

- Patch at **package**, **class**, or **function** granularity
- Patch exported functions, private functions, and struct methods
- Thread-safe patching via stop-the-world (STW) mechanism
- Support for generic functions (Go 1.18+)
- Linux (amd64, arm64) and macOS (amd64, arm64/Apple Silicon)

## Quick Start

```bash
# Build your application with inlining disabled
go build -gcflags="all=-l -N" -o myapp .

# Run it
./myapp

# Build a patch plugin from the fixed source
go build -gcflags="all=-l -N" -buildmode=plugin -o patch_v1.so .

# Apply the patch (via API call, HTTP handler, etc.)
```

## Usage

### Basic

```go
import "github.com/go-hotfix/hotfix"

// Patch specific functions
result := hotfix.Hotfix("patch_v1.so", hotfix.Func(
    "myapp/service.CalcPrice",
    "myapp/service.(*Order).Total",
))

// Patch all methods of a struct
result := hotfix.Hotfix("patch_v1.so", hotfix.Classes(
    "myapp/service.Order",        // value receiver methods
    "*myapp/service.Order",       // pointer receiver methods
))

// Patch all functions in a package
result := hotfix.Hotfix("patch_v1.so", hotfix.Package("myapp/service"))

// Combine multiple pickers
result := hotfix.Hotfix("patch_v1.so", hotfix.Any(
    hotfix.Func("myapp/service.CalcPrice"),
    hotfix.Package("myapp/util"),
))
```

### Result

```go
result := hotfix.Hotfix("patch.so", picker)
if result.Err != nil {
    log.Fatal(result.Err)
}
fmt.Printf("patched %d functions in %s\n", len(result.Methods), result.Cost)
fmt.Println(result.Message) // debug log
```

### Custom Patcher

```go
// Use the default gomonkey-based patcher
result := hotfix.DoHotfix("patch.so", picker, hotfix.GoMonkey())

// Or provide your own FuncPatcher implementation
result := hotfix.DoHotfix("patch.so", picker, myCustomPatcher)
```

## Example

A complete web application example is in [`example/webapp`](./example/webapp), demonstrating:

- A running HTTP server with a bug in `calcDiscount`
- Building a fixed plugin
- Applying the hotfix at runtime
- Verifying the fix without restarting the server

## Limitations

- **Platforms:** Linux and macOS only (due to `go plugin` constraints)
- **No closure patching:** Logic requiring hotfixes must not reside in closures
- **No signature changes:** Cannot modify data structures or function signatures — use only for bug fixes
- **Build flags:** The target program must be compiled with `-gcflags="all=-l -N"` (disable inlining and optimizations)
- **Environment consistency:** The plugin must be built with the same Go compiler version, build flags, and dependencies as the main program
- **No unloading:** Loaded plugins cannot be unloaded; excessive patching may increase memory usage
- **Plugin uniqueness:** Each plugin's `main` package must differ from previously loaded ones. Use `-ldflags="-X main.HotfixVersion=v1.0.1"` to ensure uniqueness
- **Init execution:** The plugin's `main` package `init` functions run once on load — avoid duplicate initialization

## Testing

```bash
go test -gcflags="all=-l" -v ./...
```

## Acknowledgments

Inspired by [lsg2020/go-hotfix](https://github.com/lsg2020/go-hotfix).

## License

Apache License 2.0
