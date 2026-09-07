package security

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrModuleSelfBuiltSecurity = errors.New("module has self-built security capability - must use internal/platform/security")
)

var forbiddenSecurityImports = []string{
	"crypto/tls",
	"crypto/x509",
	"golang.org/x/oauth2",
	"github.com/golang-jwt/jwt",
}

type SecurityLintResult struct {
	File       string
	Violation  string
	ImportPath string
}

func isSecurityPackage(content string) bool {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "package ") {
			return strings.TrimSpace(line[8:]) == "security"
		}
	}
	return false
}

func checkForbiddenImports(content string) []string {
	var found []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	inImportBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "import (" {
			inImportBlock = true
			continue
		}
		if inImportBlock && line == ")" {
			inImportBlock = false
			continue
		}

		var importPath string
		if inImportBlock {
			if strings.HasPrefix(line, "\"") {
				importPath = strings.Trim(line, "\"")
			}
		} else if strings.HasPrefix(line, "import ") {
			rest := strings.TrimSpace(line[7:])
			if strings.HasPrefix(rest, "\"") {
				importPath = strings.Trim(rest, "\"")
			}
		}

		if importPath == "" {
			continue
		}
		for _, forbidden := range forbiddenSecurityImports {
			if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
				found = append(found, forbidden)
			}
		}
	}
	return found
}

func LintModuleSecurity(rootDir string) ([]SecurityLintResult, error) {
	var violations []SecurityLintResult

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" || name == ".codeartsdoer" || name == "tools" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)

		if isSecurityPackage(content) {
			return nil
		}

		cleanPath := filepath.ToSlash(path)

		found := checkForbiddenImports(content)
		for _, forbidden := range found {
			violations = append(violations, SecurityLintResult{
				File:       cleanPath,
				Violation:  "imports forbidden security package",
				ImportPath: forbidden,
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("lint walk error: %w", err)
	}

	return violations, nil
}

func LintModuleSecurityStrict(rootDir string) error {
	violations, err := LintModuleSecurity(rootDir)
	if err != nil {
		return err
	}
	if len(violations) > 0 {
		msgs := make([]string, len(violations))
		for i, v := range violations {
			msgs[i] = fmt.Sprintf("  %s: %s (import: %s)", v.File, v.Violation, v.ImportPath)
		}
		return fmt.Errorf("%w:\n%s", ErrModuleSelfBuiltSecurity, strings.Join(msgs, "\n"))
	}
	return nil
}
