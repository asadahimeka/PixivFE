// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

// xxx: might have unicode byte offset problems

package main

import "strings"

func split(m Match) []Match {
	input := m.Msg

	// Step 1: Split by newline
	parts := strings.Split(input, "\n")

	var results []Match

	offset := m.Offset

	for _, part := range parts {
		// Trim whitespace
		trimmed := strings.TrimSpace(part)

		// If the trimmed string is not empty, add it to the results
		if trimmed != "" {
			results = append(results, Match{
				File:   m.File,
				Msg:    trimmed,
				Offset: offset + strings.Index(part, trimmed), // Calculate the offset of the trimmed part
			})
		}

		// Update the offset for the next part
		offset += len(part) + 1 // +1 for the newline character
	}

	return results
}
