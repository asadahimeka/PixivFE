// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
This file implements a crawler for extracting translatable strings from HTML templates.

It scans HTML files in the assets/views directory,
extracts text content while preserving some HTML structure,
and outputs the results as JSON.
*/
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Match struct {
	File   string `json:"file"`
	Msg    string `json:"msg"`
	Offset int    `json:"offset"`
}

func main() {
	result := []Match{}

	err := filepath.WalkDir("assets/views", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".html") {
			if path == "assets/views/temp.jet.html" {
				return nil
			}
			processFile(path, &result)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	encoder.Encode(result)
}

var (
	re_command_fullmatch = regexp.MustCompile(`\A([\s\n# =]*\{\{[^\{\}]*\}\})*[\s\n]*\z`)
	re_comment           = regexp.MustCompile(`\{\*[\s\S]*?\*\}`)
)

func stripJetComments(s string) string {
	return re_comment.ReplaceAllString(s, "")
}

func processFile(filename string, result *[]Match) {
	content, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	content2 := stripJetComments(string(content))

	//nolint:wsl
	findAllSimple(filename, content2, func(m Match) {
		// for _, m := range split(unsplit) {
		if !re_command_fullmatch.MatchString(m.Msg) && !shouldIgnore(m.Msg) {
			*result = append(*result, m)
		}
		// }
	})
}

// shouldIgnore is a manual filter.
func shouldIgnore(msg string) bool {
	return IgnoreTheseStrings[msg]
}
