package main

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "nopanic",
	Doc:      "forbid use of panic, log.Fatal, and os.Exit outside main function",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Пропускаем тестовые файлы
	for _, file := range pass.Files {
		filename := pass.Fset.File(file.Pos()).Name()
		if strings.HasSuffix(filename, "_test.go") {
			return nil, nil
		}
	}

	// Пропускаем vendor и другие служебные директории
	if isSkippedPackage(pass.Pkg.Path()) {
		return nil, nil
	}

	// Если TypesInfo не доступен, пропускаем анализ
	if pass.TypesInfo == nil {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Находим все объявления функций
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)

		// Проверяем, является ли функция main
		isMainFunction := pass.Pkg.Name() == "main" && fn.Name.Name == "main"

		// Проверяем все вызовы внутри этой функции
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				checkCallExpr(pass, call, isMainFunction)
			}
			return true
		})
	})

	return nil, nil
}

func checkCallExpr(pass *analysis.Pass, call *ast.CallExpr, isMainFunction bool) {
	// Проверяем panic - всегда запрещено
	if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
		pass.Reportf(call.Pos(), "use of built-in panic function is forbidden")
		return
	}

	// Проверяем log.Fatal или os.Exit с использованием TypesInfo
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		// Используем TypesInfo для получения информации о пакете
		if obj, ok := pass.TypesInfo.Uses[sel.Sel]; ok {
			if pkg, ok := obj.(*types.Func); ok {
				pkgPath := pkg.Pkg().Path()
				funcName := pkg.Name()

				// Проверяем, является ли это log.Fatal* или os.Exit
				if (pkgPath == "log" && strings.HasPrefix(funcName, "Fatal")) ||
					(pkgPath == "os" && funcName == "Exit") {

					// Разрешаем только если это вызов в функции main
					if !isMainFunction {
						pass.Reportf(call.Pos(), "call to %s.%s forbidden outside main function", pkgPath, funcName)
					}
				}
			}
		}
	}
}

func isSkippedPackage(pkgPath string) bool {
	skipPatterns := []string{
		"vendor",
		"testdata",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(pkgPath, pattern) {
			return true
		}
	}
	return false
}
