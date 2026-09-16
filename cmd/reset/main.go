package main

import (
	"go/ast"
	"go/token"
	"log"
	"os"
)

type StructInfo struct {
	PkgPath    string
	PkgName    string
	StructName string
	StructType *ast.StructType
}

func main() {
	// Получаем путь к проверяемой директории.
	// Директория передаётся в 1-м аргументе при запуске (по умолчанию - текущая директория)
	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}

	// Создаём АСТ и получаем список директорий, содержащих .go файлы
	files, err := getGolangDirs(rootDir)
	if err != nil {
		log.Fatal(err)
	}

	fset := token.NewFileSet()

	// Создаём reset.gen.go файлы
	structNames, structByPkg, err := getStructInfo(fset, files)
	if err != nil {
		log.Fatal(err)
	}

	// Создание файлов reset.gen.go
	if err = generateResetFiles(fset, structNames, structByPkg); err != nil {
		log.Fatal(err)
	}
}
