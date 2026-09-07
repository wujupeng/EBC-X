package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var domainModules = []string{
	"enterprise", "transaction", "finance", "scm", "manufacturing",
	"crm", "project", "eam", "quality", "datafabric",
	"evidence", "policy", "agent", "digitaltwin",
}

func main() {
	fset := token.NewFileSet()
	violations := []string{}

	root := "internal"
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "doc.go") {
			return nil
		}
		fromModule := extractModule(path)
		if fromModule == "" {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			toModule := extractImportModule(p)
			if toModule == "" || toModule == fromModule {
				continue
			}
			violations = append(violations,
				fmt.Sprintf("%s: %s -> %s (cross-domain direct import)", path, fromModule, toModule))
		}
		return nil
	})

	if len(violations) > 0 {
		fmt.Println("❌ Module boundary violations (TASK-R03):")
		for _, v := range violations {
			fmt.Println("  ", v)
		}
		os.Exit(1)
	}
	fmt.Println("✅ Module boundary check passed — 14 domain modules isolated (TASK-R03)")
}

func extractModule(path string) string {
	for _, m := range domainModules {
		if strings.Contains(path, "internal/"+m+"/") {
			return m
		}
	}
	return ""
}

func extractImportModule(importPath string) string {
	for _, m := range domainModules {
		if strings.Contains(importPath, "/internal/"+m) {
			return m
		}
	}
	return ""
}
