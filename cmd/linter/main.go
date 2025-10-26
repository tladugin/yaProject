package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(Analyzer)
}

// Объявляем analyzer здесь или убеждаемся, что он импортирован из analysis.go
