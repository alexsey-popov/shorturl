package linter

import (
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
)

// ForbiddenExprAnalyzer - Анализатор запрещенных выражений
var ForbiddenExprAnalyzer = &analysis.Analyzer{
	Name: "forbiddenexpr",
	Doc:  "Проверка отсутствия функции panic в любом пакете и методов os.Exit или log.Fatal за пределами функции main пакета main",
	Run:  run,
}

// run - Функция проверки АСТ для ForbiddenExprAnalyzer
func run(pass *analysis.Pass) (interface{}, error) {
	// Список запрещенных выражений
	forbiddenExpr := []string{
		"log.Fatal",
		"os.Exit",
	}

	// Проверка отсутствия методов выхода из программы (os.Exit || log.Fatal)
	findForbiddenMethod := func(x *ast.CallExpr) {
		selExpr, ok := x.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		obj := pass.TypesInfo.Uses[selExpr.Sel]
		if obj == nil {
			return
		}

		fn, ok := obj.(*types.Func)
		if !ok {
			return
		}

		if fullName := fn.FullName(); slices.Contains(forbiddenExpr, fullName) {
			pass.Reportf(selExpr.Pos(), "найден вызов запрещённого метода %s", fullName)
		}
	}

	findPanic := func(x *ast.CallExpr) {
		ident, ok := x.Fun.(*ast.Ident)
		if !ok {
			return
		}

		obj := pass.TypesInfo.Uses[ident]
		if obj == nil {
			return
		}

		if builtin, ok := obj.(*types.Builtin); ok && builtin.Name() == "panic" {
			pass.Reportf(ident.Pos(), "найден ручной вызов паники")
		}
	}

	for _, file := range pass.Files {
		// Название пакета в файле
		packageName := file.Name.Name

		// Флаг, указывающий, что мы находимся внутри func main()
		inMainFunc := false

		// функцией ast.Inspect проходим по всем узлам AST
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.FuncDecl:
				// Если это функция с именем "main" и у нее нет ресивера (это не метод структуры)
				if x.Name.Name == "main" && x.Recv == nil {
					inMainFunc = true
				} else {
					inMainFunc = false
				}
			case *ast.CallExpr: // Вызов функции или метода.
				findPanic(x) // Ищем наличие паники не зависимо от названия пакета

				// Если это не пакет main - значит нужно проверить отсутствие функций выхода из программы
				if packageName != "main" || !inMainFunc {
					findForbiddenMethod(x)
				}
			}

			return true
		})
	}
	return nil, nil
}
