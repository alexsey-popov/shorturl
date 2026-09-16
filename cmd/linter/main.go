package main

import (
	"github.com/alexsey-popov/shorturl/internal/linter"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(linter.ForbiddenExprAnalyzer)
}
