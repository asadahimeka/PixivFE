// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Package main is a crawler to find i18n function calls in Go files.

Matches the pattern: i18n.$FUNC("$MSG", $...ARGS)
*/
package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

// Match represents an i18n function call found in the code.
type Match struct {
	File   string `json:"file"`
	Msg    string `json:"msg"`
	Line   int    `json:"line"`
	Offset int    `json:"offset"`
}

func main() {
	matches, err := findI18nCalls(".")
	if err != nil {
		log.Fatalf("Error finding i18n calls: %v", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(matches); err != nil {
		log.Fatalf("Error encoding JSON: %v", err)
	}
}

// findI18nCalls walks through all Go files in the given directory and finds i18n function calls.
func findI18nCalls(rootDir string) ([]Match, error) {
	var matches []Match
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		fileMatches, err := parseFileForI18n(path)
		if err != nil {
			log.Printf("Warning: failed to parse file %s: %v", path, err)
			return nil
		}
		matches = append(matches, fileMatches...)
		return nil
	})

	return matches, err
}

// parseFileForI18n parses a single Go file and extracts i18n function calls.
func parseFileForI18n(filePath string) ([]Match, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var matches []Match
	ast.Inspect(node, func(n ast.Node) bool {
		if match := extractI18nCall(n, fset, filePath); match != nil {
			matches = append(matches, *match)
		}
		return true
	})

	return matches, nil
}

// extractI18nCall checks if the AST node is an i18n function call and extracts the match.
func extractI18nCall(n ast.Node, fset *token.FileSet, filePath string) *Match {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "i18n" {
		return nil
	}
	if len(call.Args) == 0 {
		return nil
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return nil
	}
	msg, err := strconv.Unquote(lit.Value)
	if err != nil {
		return nil
	}
	pos := fset.Position(call.Lparen)
	return &Match{File: filePath, Msg: msg, Line: pos.Line, Offset: pos.Offset}
}
