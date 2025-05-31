// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package main

// IgnoreTheseStrings is a map of strings that should be ignored by the i18n crawler,
// such as those that remain unchanged across all locales.
var IgnoreTheseStrings = map[string]bool{
	"":                      true,
	"»":                     true,
	"▶":                     true,
	"⧉ {{ .Pages }}":        true,
	"PixivFE":               true,
	"pixiv.net/i/{{ .ID }}": true,
}
