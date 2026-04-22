package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestExtractGoImports_SingleAndBlock(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "main.go", `package main

import "fmt"
import "strings"

import (
    "os"
    aliased "path/filepath"
    "github.com/example/lib"
)

func main() {}
`)
	graph, err := ExtractImportGraph([]string{path})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	imports := graph[path]
	want := []string{"fmt", "strings", "os", "path/filepath", "github.com/example/lib"}
	if len(imports) != len(want) {
		t.Fatalf("want %d imports, got %d: %v", len(want), len(imports), imports)
	}
	for _, w := range want {
		found := false
		for _, got := range imports {
			if got == w {
				found = true
			}
		}
		if !found {
			t.Errorf("missing import %q in %v", w, imports)
		}
	}
}

func TestExtractTSImports(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "app.ts", `import { Foo } from './foo';
import Bar from "../bar/index";
import * as baz from 'baz-lib';

export { something } from './exports';
const mod = await import('./lazy');
`)
	graph, err := ExtractImportGraph([]string{path})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	imports := graph[path]
	want := []string{"./foo", "../bar/index", "baz-lib", "./lazy", "./exports"}
	for _, w := range want {
		found := false
		for _, got := range imports {
			if got == w {
				found = true
			}
		}
		if !found {
			t.Errorf("missing TS import %q in %v", w, imports)
		}
	}
}

func TestExtractPyImports(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "mod.py", `import os
import sys
from pathlib import Path
from .relative import thing
from foo.bar import baz
`)
	graph, err := ExtractImportGraph([]string{path})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	imports := graph[path]
	// Expect: os, sys, pathlib, foo.bar (the `.relative` form isn't
	// captured by the simple regex — that's acceptable; we match module
	// names, not relative resolution).
	want := []string{"os", "sys", "pathlib", "foo.bar"}
	for _, w := range want {
		found := false
		for _, got := range imports {
			if got == w {
				found = true
			}
		}
		if !found {
			t.Errorf("missing py import %q in %v", w, imports)
		}
	}
}

func TestExtractImportGraph_MissingFile(t *testing.T) {
	graph, err := ExtractImportGraph([]string{"/nope/nowhere.go"})
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if len(graph) != 0 {
		t.Fatalf("missing file should not appear, got %v", graph)
	}
}

func TestRenderMermaid_IsDeterministic(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.go", "package a\nimport \"fmt\"\n")
	b := writeFile(t, dir, "b.go", "package b\nimport \"fmt\"\n")
	graph, _ := ExtractImportGraph([]string{b, a}) // reversed
	out := graph.RenderMermaid()
	if !strings.Contains(out, "flowchart LR") {
		t.Errorf("expected flowchart header, got %q", out)
	}
	// a.go should appear before b.go regardless of input order
	if strings.Index(out, "a_go") > strings.Index(out, "b_go") {
		t.Errorf("not alphabetical: a_go should precede b_go in %q", out)
	}
}
