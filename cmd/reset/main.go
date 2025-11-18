package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func main() {
	// Определяем корневую директорию проекта
	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	} else {
		// Пытаемся найти корень проекта автоматически
		if dir, err := findProjectRoot(); err == nil {
			rootDir = dir
		}
	}

	if err := run(rootDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return ".", err
	}

	// Ищем go.mod файл
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Достигли корневой директории
		}
		dir = parent
	}

	return ".", fmt.Errorf("go.mod not found")
}

func run(rootDir string) error {
	// Собираем информацию о всех пакетах
	packages := make(map[string]*PackageInfo)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		// Пропускаем директории, начинающиеся с . (например, .git, .vscode)
		if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
			return filepath.SkipDir
		}

		// Пропускаем директорию cmd/reset чтобы не обрабатывать саму утилиту
		if strings.Contains(path, "cmd/reset") {
			return filepath.SkipDir
		}

		// Пропускаем vendor и testdata
		if d.Name() == "vendor" || d.Name() == "testdata" {
			return filepath.SkipDir
		}

		// Анализируем Go файлы в директории
		fset := token.NewFileSet()
		files, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		pkgInfo := &PackageInfo{
			Path:    path,
			Structs: make([]StructInfo, 0),
			FileSet: fset,
			Files:   make([]*ast.File, 0),
		}

		// Парсим каждый Go файл в директории
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			name := file.Name()
			// Пропускаем тестовые файлы и сгенерированные файлы
			if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".gen.go") {
				continue
			}

			if !strings.HasSuffix(name, ".go") {
				continue
			}

			filename := filepath.Join(path, name)
			parsedFile, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
			if err != nil {
				continue
			}

			pkgInfo.Files = append(pkgInfo.Files, parsedFile)
			if pkgInfo.Name == "" {
				pkgInfo.Name = parsedFile.Name.Name
			}
		}

		if len(pkgInfo.Files) == 0 {
			return nil
		}

		// Ищем структуры с комментарием generate:reset
		if err := pkgInfo.findResetableStructs(); err != nil {
			return err
		}

		if len(pkgInfo.Structs) > 0 {
			packages[path] = pkgInfo
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("walking directory: %w", err)
	}

	if len(packages) == 0 {
		fmt.Println("No packages with resetable structs found")
		return nil
	}

	// Генерируем файлы reset.gen.go для каждого пакета
	for path, pkgInfo := range packages {
		if err := pkgInfo.generateResetFile(); err != nil {
			return fmt.Errorf("generating reset file for %s: %w", path, err)
		}
		fmt.Printf("Generated reset.gen.go for package %s with %d methods\n",
			pkgInfo.Name, len(pkgInfo.Structs))
	}

	return nil
}

type PackageInfo struct {
	Name    string
	Path    string
	Structs []StructInfo
	FileSet *token.FileSet
	Files   []*ast.File
}

type StructInfo struct {
	Name   string
	Fields []FieldInfo
}

type FieldInfo struct {
	Name      string
	Type      string
	IsPointer bool
	IsSlice   bool
	IsMap     bool
	IsStruct  bool
}

func (pkg *PackageInfo) findResetableStructs() error {
	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.GenDecl:
				if x.Tok == token.TYPE {
					for _, spec := range x.Specs {
						typeSpec := spec.(*ast.TypeSpec)
						structType, ok := typeSpec.Type.(*ast.StructType)
						if !ok {
							continue
						}

						// Проверяем комментарий generate:reset
						if hasGenerateResetComment(x) {
							structInfo := StructInfo{
								Name:   typeSpec.Name.Name,
								Fields: pkg.extractFieldInfo(structType),
							}
							pkg.Structs = append(pkg.Structs, structInfo)
						}
					}
				}
			}
			return true
		})
	}
	return nil
}

func hasGenerateResetComment(decl *ast.GenDecl) bool {
	if decl.Doc == nil {
		return false
	}

	for _, comment := range decl.Doc.List {
		if strings.Contains(comment.Text, "generate:reset") {
			return true
		}
	}
	return false
}

func (pkg *PackageInfo) extractFieldInfo(structType *ast.StructType) []FieldInfo {
	var fields []FieldInfo

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue // Пропускаем анонимные поля
		}

		fieldName := field.Names[0].Name
		if !unicode.IsUpper(rune(fieldName[0])) {
			continue // Пропускаем приватные поля
		}

		fieldInfo := FieldInfo{
			Name: fieldName,
		}

		// Анализируем тип поля
		fieldInfo.Type, fieldInfo.IsPointer, fieldInfo.IsSlice, fieldInfo.IsMap, fieldInfo.IsStruct =
			pkg.analyzeType(field.Type)

		fields = append(fields, fieldInfo)
	}

	return fields
}

func (pkg *PackageInfo) analyzeType(expr ast.Expr) (typeName string, isPointer, isSlice, isMap, isStruct bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		typeName = t.Name
		isStruct = isBuiltinStructType(typeName)

	case *ast.StarExpr:
		typeName, _, isSlice, isMap, isStruct = pkg.analyzeType(t.X)
		isPointer = true

	case *ast.ArrayType:
		typeName, _, _, _, isStruct = pkg.analyzeType(t.Elt)
		isSlice = true
		typeName = "[]" + typeName

	case *ast.MapType:
		keyType, _, _, _, _ := pkg.analyzeType(t.Key)
		valueType, _, _, _, _ := pkg.analyzeType(t.Value)
		typeName = "map[" + keyType + "]" + valueType
		isMap = true

	case *ast.SelectorExpr:
		// Для типов из других пакетов
		pkgName := t.X.(*ast.Ident).Name
		typeName = pkgName + "." + t.Sel.Name
		isStruct = true // Предполагаем, что это структура

	default:
		typeName = fmt.Sprintf("%T", t)
	}

	return
}

func isBuiltinStructType(typeName string) bool {
	builtinTypes := map[string]bool{
		"int": false, "int8": false, "int16": false, "int32": false, "int64": false,
		"uint": false, "uint8": false, "uint16": false, "uint32": false, "uint64": false,
		"float32": false, "float64": false, "complex64": false, "complex128": false,
		"string": false, "bool": false, "byte": false, "rune": false,
		"error": false,
	}

	_, isBuiltin := builtinTypes[typeName]
	return !isBuiltin
}

func (pkg *PackageInfo) generateResetFile() error {
	var buf bytes.Buffer

	// Заголовок файла
	buf.WriteString("// Code generated by reset. DO NOT EDIT.\n")
	buf.WriteString("package " + pkg.Name + "\n\n")

	// Генерируем методы Reset для каждой структуры
	for _, structInfo := range pkg.Structs {
		buf.WriteString(pkg.generateResetMethod(structInfo))
		buf.WriteString("\n")
	}

	// Форматируем код
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting source: %w\nSource: %s", err, buf.String())
	}

	// Записываем файл
	resetFilePath := filepath.Join(pkg.Path, "reset.gen.go")
	return os.WriteFile(resetFilePath, formatted, 0644)
}

func (pkg *PackageInfo) generateResetMethod(structInfo StructInfo) string {
	var buf bytes.Buffer

	buf.WriteString("func (rs *" + structInfo.Name + ") Reset() {\n")
	buf.WriteString("    if rs == nil {\n")
	buf.WriteString("        return\n")
	buf.WriteString("    }\n\n")

	for _, field := range structInfo.Fields {
		buf.WriteString(pkg.generateFieldReset(field))
	}

	buf.WriteString("}\n")
	return buf.String()
}

func (pkg *PackageInfo) generateFieldReset(field FieldInfo) string {
	var buf bytes.Buffer

	switch {
	case field.IsSlice:
		buf.WriteString("    rs." + field.Name + " = rs." + field.Name + "[:0]\n")

	case field.IsMap:
		buf.WriteString("    clear(rs." + field.Name + ")\n")

	case field.IsPointer && field.IsStruct:
		// Для указателей на структуры проверяем наличие метода Reset
		buf.WriteString("    if rs." + field.Name + " != nil {\n")
		buf.WriteString("        if resetter, ok := interface{}(rs." + field.Name + ").(interface{ Reset() }); ok {\n")
		buf.WriteString("            resetter.Reset()\n")
		buf.WriteString("        }\n")
		buf.WriteString("    }\n")

	case field.IsPointer:
		// Для указателей на примитивы сбрасываем значение
		buf.WriteString("    if rs." + field.Name + " != nil {\n")
		buf.WriteString("        *rs." + field.Name + " = " + getZeroValue(field.Type) + "\n")
		buf.WriteString("    }\n")

	case field.IsStruct:
		// Для вложенных структур вызываем Reset если есть
		buf.WriteString("    if resetter, ok := interface{}(&rs." + field.Name + ").(interface{ Reset() }); ok {\n")
		buf.WriteString("        resetter.Reset()\n")
		buf.WriteString("    }\n")

	default:
		// Для примитивных типов
		buf.WriteString("    rs." + field.Name + " = " + getZeroValue(field.Type) + "\n")
	}

	return buf.String()
}

func getZeroValue(typeName string) string {
	switch typeName {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "complex64", "complex128":
		return "0"
	case "string":
		return `""`
	case "bool":
		return "false"
	default:
		return typeName + "{}"
	}
}
