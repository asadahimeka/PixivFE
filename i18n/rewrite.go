// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
This file rewrites strings in Jet templates before they are parsed.
*/
package i18n

import (
	"slices"
	"strings"
)

var tm = map[cacheKey]*strings.Replacer{}

// format: old0, new0, old1, new1, ...
type TrPairs = []string

type cacheKey = struct {
	locale string
	file   string
}

type translationPair = struct {
	before string
	after  string
}

// func RewriteString(file string, content string) string {
// 	locale := getLocale()
// 	if locale == BaseLocale {
// 		return content
// 	}
// 	replacer := TranslationReplacer(locale, file)
// 	return replacer.Replace(content)
// }

// Replacer returns nil when nothing needs to be replaced.
func Replacer(locale string, file string) *strings.Replacer {
	if locale == BaseLocale {
		return nil
	}

	k := cacheKey{locale: locale, file: file}

	v, exist := tm[k]
	if exist {
		return v
	}

	pairs := translationPairsInner(locale, file)
	if len(pairs) == 0 {
		v = nil
	} else {
		v = strings.NewReplacer(pairs...)
	}

	tm[k] = v

	return v
}

func translationPairsInner(locale string, file string) TrPairs {
	fromMap := locales[BaseLocale]

	toMap, exist := locales[locale]
	if !exist {
		return TrPairs{}
	}

	staging := []translationPair{}

	for k, v := range toMap {
		if strings.HasPrefix(k, file+":") && fromMap[k] != v {
			staging = append(staging, translationPair{fromMap[k], v})
		}
	}

	// sort by length. longest first.
	// this is to prevent weird stuff when rewriting multiple strings and
	// a short one is substring of a long one.
	slices.SortFunc(staging, func(a translationPair, b translationPair) int {
		return len(b.before) - len(a.before)
	})

	result := TrPairs{}
	for _, v := range staging {
		if len(v.before) < 1 {
			continue
		}

		if strings.Count(v.before, " ") < 1 {
			result = append(result, ">"+v.before)
			result = append(result, ">"+v.after)
			result = append(result, "\t"+v.before)
			result = append(result, "\t"+v.after)
			result = append(result, "\""+v.before)
			result = append(result, "\""+v.after)
			result = append(result, " "+v.before)
			result = append(result, " "+v.after)

			continue
		}

		result = append(result, v.before)
		result = append(result, v.after)
	}

	return result
}
