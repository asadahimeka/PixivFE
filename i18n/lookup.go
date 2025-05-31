// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
This file loads translation files and looks up translations for given strings.
*/
package i18n

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"path"
	"runtime"

	"codeberg.org/pixivfe/pixivfe/server/assets"
	"github.com/zeebo/xxh3"
)

const (
	SuccintIDBytes int = 8
	// CallerSkipFrames is the number of stack frames to skip to reach the calling function
	// that invokes the translation lookup.
	CallerSkipFrames int = 2
)

var locales = map[string]map[string]string{}

func Setup() error {
	fsI18n, err := fs.Sub(assets.FS, "i18n/locale")
	if err != nil {
		return err
	}

	entries, err := fs.ReadDir(fsI18n, ".")
	if err != nil {
		return fmt.Errorf("failed to read i18n directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.Type().IsDir() {
			continue
		}

		locales[entry.Name()], err = loadLocale(fsI18n, entry.Name())
		if err != nil {
			return err
		}

		log.Printf("Loaded locale %s", entry.Name())
	}

	return nil
}

func loadLocale(fsI18n fs.FS, locale string) (map[string]string, error) {
	codeTranslations, err := loadLocaleHelper(fsI18n, locale, "code.json")
	if err != nil {
		return nil, err
	}

	templateTranslations, err := loadLocaleHelper(fsI18n, locale, "template.json")
	if err != nil {
		return nil, err
	}

	maps.Copy(codeTranslations, templateTranslations)

	return codeTranslations, nil
}

func loadLocaleHelper(fsI18n fs.FS, locale string, filename string) (map[string]string, error) {
	// path.Join will create paths like "en/code.json", relative to fsI18n
	// which itself is already "i18n/locale" from assets.FS
	filePath := path.Join(locale, filename)
	file, err := fsI18n.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open translation file %s/%s: %w", locale, filename, err)
	}
	defer file.Close()

	var translatedStrings map[string]string

	err = json.NewDecoder(file).Decode(&translatedStrings)
	if err != nil {
		return nil, fmt.Errorf("failed to decode translation file %s/%s: %w", locale, filename, err)
	}

	return translatedStrings, nil
}

// lookupSkipStack2 lookup string in the translation database.
//
// call stack should look like this: caller (in the correct file) -> another function -> lookupSkipStack2
//
// For internal use only; do not call this function directly.
func lookupSkipStack2(locale string, text string) string {
	if locale == BaseLocale {
		return text
	}

	translationMap, exist := locales[locale]
	if !exist {
		return text
	}

	_, file, _, ok := runtime.Caller(CallerSkipFrames) // user function -> i18n.XXX("...") -> lookup
	// file and line is correct
	// println("recorded stackframe:", pc, file, line, ok, text)
	if !ok {
		return text
	}

	translation, exist := translationMap[SuccintID(file, text)]
	if !exist {
		return text
	}

	return translation
}

func SuccintID(file string, text string) string {
	hash := xxh3.HashString(text)
	hashBytes := make([]byte, SuccintIDBytes)
	binary.LittleEndian.PutUint64(hashBytes, hash)
	digest := base64.RawURLEncoding.EncodeToString(hashBytes)

	return file + ":" + digest
}
