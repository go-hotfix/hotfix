package hotfix

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-hotfix/assembly"
)

// closureExp matches closure identifiers (e.g. "func1", "func1.2") that
// cannot be hot-patched.
var closureExp = regexp.MustCompile(`func\d+(\.\d+)*`)

// FuncPicker selects which functions should be hot-patched. Given a
// DwarfAssembly for runtime introspection, it returns the fully qualified
// names of the target functions.
type FuncPicker func(dwarfAssembly assembly.DwarfAssembly) ([]string, error)

// Func returns a FuncPicker that selects specific functions by their fully
// qualified names. Names must use the Go runtime format:
//
//	example/data.TestAdd
//	example/data.(*DataType).TestHotfix
//	example/data.testPrivateFunc
func Func(funcNames ...string) FuncPicker {
	return func(_ assembly.DwarfAssembly) ([]string, error) {
		for _, name := range funcNames {
			if closureExp.MatchString(name) {
				return nil, fmt.Errorf("closure unsupported: %s", name)
			}
		}
		return funcNames, nil
	}
}

// Classes returns a FuncPicker that selects all methods of the specified
// struct types. The className must be the fully qualified type name:
//
//	example/data.DataType        — value receiver methods
//	*example/data.DataType       — pointer receiver methods
func Classes(classNames ...string) FuncPicker {
	return func(dwarfAssembly assembly.DwarfAssembly) ([]string, error) {
		var methods []string
		for _, className := range classNames {
			classType, err := dwarfAssembly.FindType(className)
			if nil != err {
				return nil, fmt.Errorf("%w: class not found: %s", err, className)
			}

			if reflect.Struct != classType.Kind() && (reflect.Ptr != classType.Kind() || classType.Elem().Kind() != reflect.Struct) {
				return nil, fmt.Errorf("%s is not a struct or *struct (%s)", className, classType.String())
			}

			isPtr := reflect.Ptr == classType.Kind()
			if isPtr {
				classType = classType.Elem()
			}

			var prefixName string
			if isPtr {
				prefixName = classType.PkgPath() + ".(*" + classType.Name() + ")."
			} else {
				prefixName = classType.PkgPath() + "." + classType.Name() + "."
			}

			for name := range dwarfAssembly.Funcs() {
				if strings.HasPrefix(name, prefixName) && !closureExp.MatchString(name) {
					methods = append(methods, name)
				}
			}
		}

		return methods, nil
	}
}

// Package returns a FuncPicker that selects all functions and methods
// belonging to the specified package(s). The pkg must be the full
// import path:
//
//	example/data
func Package(pkgs ...string) FuncPicker {
	return func(dwarfAssembly assembly.DwarfAssembly) ([]string, error) {
		var methods []string
		for _, pkg := range pkgs {
			var prefixName = pkg

			for name := range dwarfAssembly.Funcs() {
				if strings.HasPrefix(name, prefixName) && !closureExp.MatchString(name) {
					methods = append(methods, name)
				}
			}
		}
		return methods, nil
	}
}

// Any combines multiple FuncPickers into one. The selected function names
// are concatenated in order. If any picker returns an error, the combined
// picker returns that error immediately.
func Any(funcPickers ...FuncPicker) FuncPicker {
	return func(dwarfAssembly assembly.DwarfAssembly) ([]string, error) {
		var methods []string
		for _, picker := range funcPickers {
			mm, err := picker(dwarfAssembly)
			if nil != err {
				return nil, err
			}
			methods = append(methods, mm...)
		}
		return methods, nil
	}
}
