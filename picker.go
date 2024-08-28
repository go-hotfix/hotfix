package hotfix

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-hotfix/assembly"
)

var closureExp = regexp.MustCompile(`func\d+(\.\d+)*`)

// FuncPicker List of functions that need to be hotfix.
type FuncPicker func(dwarfAssembly assembly.DwarfAssembly) ([]string, error)

// Func to specify one or more functions, you must use the full qualified name of the function.
//
//	example/data.TestAdd
//	example/data.(*DataType).TestHotfix
//	example/data.testPrivateFunc
//	example/data.(*DataType).test
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

// Classes To fix a specified class or classes (all member functions), the fully qualified name of the class must be used.
//
//	example/data.DataType
//	*example/data.DataType
func Classes(classNames ...string) FuncPicker {
	return func(dwarfAssembly assembly.DwarfAssembly) ([]string, error) {
		var methods []string
		for _, className := range classNames {
			// 查找类型
			classType, err := dwarfAssembly.FindType(className)
			if nil != err {
				return nil, fmt.Errorf("%w: class not found: %s", err, className)
			}

			// 检查类型必须是 struct/*struct
			if reflect.Struct != classType.Kind() && (reflect.Ptr != classType.Kind() || classType.Elem().Kind() != reflect.Struct) {
				return nil, fmt.Errorf("%s is not a struct or *struct (%s)", className, classType.String())
			}

			isPtr := reflect.Ptr == classType.Kind()
			if isPtr {
				classType = classType.Elem()
			}

			var prefixName string
			if isPtr {
				// example.data.(*DataType).String
				prefixName = classType.PkgPath() + ".(*" + classType.Name() + ")."
			} else {
				// example.data.DataType.String
				prefixName = classType.PkgPath() + "." + classType.Name() + "."
			}

			dwarfAssembly.ForeachFunc(func(name string, pc uint64) bool {
				if strings.HasPrefix(name, prefixName) && !closureExp.MatchString(name) {
					methods = append(methods, name)
				}
				return true
			})
		}

		return methods, nil
	}
}

// Package To fix all export, private, and member functions in one or more packages, the full package name must be used
//
//	example/data
func Package(pkgs ...string) FuncPicker {
	return func(dwarfAssembly assembly.DwarfAssembly) ([]string, error) {
		var methods []string
		for _, pkg := range pkgs {

			// example/data.testPrivateFunc
			// example/data.(*DataType).TestHotfix
			var prefixName = pkg

			dwarfAssembly.ForeachFunc(func(name string, pc uint64) bool {
				if strings.HasPrefix(name, prefixName) && !closureExp.MatchString(name) {
					methods = append(methods, name)
				}
				return true
			})
		}
		return methods, nil
	}
}

// Any combine multiple FuncPicker
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
