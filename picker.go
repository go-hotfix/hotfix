package hotfix

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-hotfix/assembly"
)

var closureExp = regexp.MustCompile(`func\d+(\.\d+)*`)

type FuncPicker func(dwarfAssembly assembly.DwarfAssembly) ([]string, error)

// Func 修复指定一个或者多个函数，必须使用函数的完整限定名
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

// Classes 修复指定的一个或者多个类（所有成员函数），必须使用类的完整限定名
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

// Package 修复指定的一个或者多个包（所有非/成员函数），必须使用完整包名
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

// Any 组合多种方式
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
