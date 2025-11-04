package main

import (
	"go/ast"
	"go/token"
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
	// Пропускаем тестовые файлы - исправленная логика
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

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	isMainPkg := pass.Pkg.Name() == "main"

	// Создаем карту для отслеживания функций main
	mainFuncRanges := make(map[string]token.Pos)

	// Сначала находим все функции main и их позиции конца
	insp.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Name.Name == "main" {
			// Сохраняем конечную позицию функции main
			mainFuncRanges[pass.Fset.File(fn.Pos()).Name()] = fn.End()
		}
	})

	// Теперь проверяем все вызовы функций
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		filename := pass.Fset.File(call.Pos()).Name()

		// Проверяем, находится ли вызов в main функции
		inMainFunc := false
		if mainEnd, exists := mainFuncRanges[filename]; exists {
			inMainFunc = call.Pos() < mainEnd
		}

		checkCallExpr(pass, call, isMainPkg, inMainFunc)
	})

	return nil, nil
}

func checkCallExpr(pass *analysis.Pass, call *ast.CallExpr, isMainPkg bool, inMainFunc bool) {
	// Check for panic - всегда запрещено
	if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
		pass.Reportf(call.Pos(), "use of built-in panic function is forbidden")
		return
	}

	// Check for log.Fatal or os.Exit
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if pkg, ok := sel.X.(*ast.Ident); ok {
			pkgName := pkg.Name
			funcName := sel.Sel.Name

			if (pkgName == "log" && strings.HasPrefix(funcName, "Fatal")) ||
				(pkgName == "os" && funcName == "Exit") {

				// Разрешаем только если это main пакет И вызов в main функции
				if !(isMainPkg && inMainFunc) {
					pass.Reportf(call.Pos(), "call to %s.%s forbidden outside main function", pkgName, funcName)
				}
			}
		}
	}
}

func isSkippedPackage(pkgPath string) bool {
	skipPatterns := []string{
		"vendor",
		"testdata", // Добавляем testdata в игнорируемые
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(pkgPath, pattern) {
			return true
		}
	}
	return false
}
