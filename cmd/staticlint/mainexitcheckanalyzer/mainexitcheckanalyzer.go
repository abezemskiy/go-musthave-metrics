// пакет mainexitcheckanalyzer представляет собой статический анализатор, который
// выявляет использование os.Exit в функции main.
package mainexitcheckanalyzer

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// MainExitCheckAnalyzer - экспортируемая переменная для использования анализатора.
var MainExitCheckAnalyzer = &analysis.Analyzer{
	Name: "mainexitcheck",
	Doc:  "check for using os.Exit in main function",
	Run:  run,
}

// isOsExitCalling - проверяет, является ли вызов функцией os.Exit.
func isOsExitCalling(pass *analysis.Pass, call *ast.CallExpr) bool {
	// Проверка, что вызов функции состоит из двух частей: os и Exit
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		// Проверяем, что имя метода — "Exit"
		if sel.Sel.Name == "Exit" {
			// Проверяем, что идентификатор — "os"
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" {
				// Проверяем, что пакет "os" импортирован
				for _, imp := range pass.Pkg.Imports() {
					if imp.Path() == "os" {
						return true
					}
				}
			}
		}
	}
	return false
}

// run - основная функция анализа, которая запускается анализатором.
func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// пропускаю файлы кэша, чтобы анализировать только исходные файлы
		filename := pass.Fset.Position(file.Pos()).Filename
		if strings.Contains(filename, "/.cache/go-build") {
			continue
		}

		// Проходим по каждому узлу AST файла
		ast.Inspect(file, func(node ast.Node) bool {
			// Проверяем, что текущий узел — это определение функции
			if fn, ok := node.(*ast.FuncDecl); ok {
				// Проверяем, что это функция main
				if fn.Name.Name == "main" {
					// Проходим по выражениям в теле функции
					for _, stmt := range fn.Body.List {
						// Проверяем, является ли выражение вызовом функции os.Exit
						if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
							if call, ok := exprStmt.X.(*ast.CallExpr); ok && isOsExitCalling(pass, call) {
								pass.Reportf(call.Pos(), "using os.Exit in main function is prohibited")
							}
						}
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
