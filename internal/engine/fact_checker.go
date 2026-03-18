package engine

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Mismatch represents a fact-check failure.
type Mismatch struct {
	Claim    string `json:"claim"`
	Type     string `json:"type"`     // file, symbol, linecount, import
	Severity string `json:"severity"` // critical, major, minor
	Detail   string `json:"detail"`
}

// VerifyResult contains the results of fact-checking a document.
type VerifyResult struct {
	File       string     `json:"file"`
	Mismatches []Mismatch `json:"mismatches"`
	Summary    Summary    `json:"summary"`
}

// Summary counts issues by severity.
type Summary struct {
	Critical int `json:"critical"`
	Major    int `json:"major"`
	Minor    int `json:"minor"`
	Info     int `json:"info"`
	Checked  int `json:"checked"`
}

// File path patterns: matches paths like src/auth/oauth.ts, internal/cli/root.go
// Both in backticks and in plain text / code blocks
var filePathRe = regexp.MustCompile(`(?:` + "`" + `)?([a-zA-Z0-9_][a-zA-Z0-9_./-]*\.[a-zA-Z0-9]{1,10})(?:` + "`" + `)?`)

// Symbol patterns (function/type/class names)
var symbolRe = regexp.MustCompile("`(?:function |class |type |interface |export (?:function |class |const |type ))([a-zA-Z_][a-zA-Z0-9_]*)`")

// VerifyFile fact-checks a markdown document against the filesystem.
func VerifyFile(docPath string, projectRoot string) (*VerifyResult, error) {
	data, err := os.ReadFile(docPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", docPath, err)
	}

	content := string(data)
	result := &VerifyResult{File: docPath}

	// Extract and check file references
	filePaths := extractFilePaths(content)
	for _, fp := range filePaths {
		result.Summary.Checked++
		absPath := filepath.Join(projectRoot, fp)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			result.Mismatches = append(result.Mismatches, Mismatch{
				Claim:    fp,
				Type:     "file",
				Severity: "critical",
				Detail:   fmt.Sprintf("File does not exist: %s", absPath),
			})
			result.Summary.Critical++
		}
	}

	// Extract and check symbol references in existing files
	symbols := extractSymbols(content)
	for _, sym := range symbols {
		if sym.File == "" {
			continue
		}
		result.Summary.Checked++
		absFile := filepath.Join(projectRoot, sym.File)
		if _, err := os.Stat(absFile); os.IsNotExist(err) {
			continue // File doesn't exist — already caught above
		}
		if !grepSymbol(absFile, sym.Name) {
			result.Mismatches = append(result.Mismatches, Mismatch{
				Claim:    fmt.Sprintf("%s in %s", sym.Name, sym.File),
				Type:     "symbol",
				Severity: "critical",
				Detail:   fmt.Sprintf("Symbol '%s' not found in %s", sym.Name, sym.File),
			})
			result.Summary.Critical++
		}
	}

	return result, nil
}

type symbolRef struct {
	Name string
	File string
}

func extractFilePaths(content string) []string {
	matches := filePathRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var paths []string

	for _, m := range matches {
		fp := m[1]
		// Filter: must have directory separator (a/b.c pattern)
		if !strings.Contains(fp, "/") {
			continue
		}
		// Skip URLs
		if strings.HasPrefix(fp, "http") {
			continue
		}
		// Skip common non-file patterns
		if strings.HasPrefix(fp, "v1.") || strings.HasPrefix(fp, "v2.") {
			continue
		}
		// Skip Go module paths (contain domain-like patterns)
		if strings.Contains(fp, ".com/") || strings.Contains(fp, ".io/") || strings.Contains(fp, ".in/") || strings.Contains(fp, ".org/") {
			continue
		}
		// Skip markdown header patterns
		if strings.HasSuffix(fp, ".md") && strings.HasPrefix(fp, "#") {
			continue
		}
		if !seen[fp] {
			seen[fp] = true
			paths = append(paths, fp)
		}
	}

	return paths
}

func extractSymbols(content string) []symbolRef {
	// Simple heuristic: look for patterns like "functionName in src/file.ts"
	// or "src/file.ts:functionName"
	lines := strings.Split(content, "\n")
	var refs []symbolRef

	colonRe := regexp.MustCompile("`([a-zA-Z0-9_./-]+\\.[a-zA-Z]+):([a-zA-Z_][a-zA-Z0-9_]*)`")
	for _, line := range lines {
		matches := colonRe.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			refs = append(refs, symbolRef{File: m[1], Name: m[2]})
		}
	}

	return refs
}

func grepSymbol(filePath, symbol string) bool {
	cmd := exec.Command("grep", "-q", symbol, filePath)
	return cmd.Run() == nil
}

// CountLines returns the line count of a file.
func CountLines(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}
