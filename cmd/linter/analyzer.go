package main

import (
	"go/ast"
	"slices"

	"golang.org/x/tools/go/analysis"
)

// ForbittenExprAnalyzer - Анализатор запрещенных выражений
var ForbittenExprAnalyzer = &analysis.Analyzer{
	Name: "forbittenexpr",
	Doc:  "Проверка отсутствия функции panic в любом пакете и методов os.Exit или log.Fatal за пределами функции main пакета main",
	Run:  run,
}

// run - Функция проверки АСТ для ForbittenExprAnalyzer
func run(pass *analysis.Pass) (interface{}, error) {
	// Список запрещенных выражений
	forbittenExpr := []string{
		"log.Fatal",
		"os.Exit",
	}

	// Проверка отсутствия методов выхода из программы (os.Exit || log.Fatal)
	findForbiddenMethod := func(x *ast.CallExpr) {
		// Проверяем, что выражение является методом, а не функцией
		selExpr, ok := x.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		// Проверяем, что метод вызывается у пакета/переменной, а не у метода
		// Например в a.b.method() параметр selExpr.X будет равен ast.SelectorExpr, а не ast.Ident
		ident, ok := selExpr.X.(*ast.Ident)
		if !ok {
			return
		}

		// Склеиваем имя метода и имя пакета/переменной и сличаем со списком запрещённых вызовов
		if expr := ident.Name + "." + selExpr.Sel.Name; slices.Contains(forbittenExpr, expr) {
			pass.Reportf(ident.Pos(), "найден вызов запрещённого метода %s", expr)
		}
	}

	findPanic := func(x *ast.CallExpr) {
		// Нам нужно отличить метод panic от функции, поэтому пытаемся преобразовать объект к типу Ident
		// (метод не сможет прийти к этому типу)
		ident, ok := x.Fun.(*ast.Ident)
		if ok && ident.Name == "panic" {
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
