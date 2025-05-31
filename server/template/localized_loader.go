// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package template

import (
	"io"
	"io/fs"
	"path"
	"strings"

	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/assets"
)

// LocalizedFSLoader implements jet.Loader to load templates from the embedded
// assets.FS and apply i18n localization.
type LocalizedFSLoader struct {
	// Dir is the slash-separated base directory for resolving template paths within assets.FS.
	Dir string
}

// Exists checks if a template file exists in assets.FS.
//
// If templatePath is absolute, l.Dir is effectively ignored in path.Join.
func (l *LocalizedFSLoader) Exists(templatePath string) bool {
	// path.Join ensures the resulting path uses slashes, suitable for fs.FS.
	finalPath := path.Join(l.Dir, templatePath)

	stat, err := fs.Stat(assets.FS, finalPath) // Check against assets.FS
	if err == nil && !stat.IsDir() {
		return true
	}

	return false
}

// Open loads a template from assets.FS, localizes it, and returns its content.
//
// If templatePath is absolute, l.Dir is effectively ignored in path.Join
// for assets.FS access and i18n resource lookup.
//
// Falls back to unlocalized content if no i18n replacer is found.
func (l *LocalizedFSLoader) Open(templatePath string) (io.ReadCloser, error) {
	locale := i18n.GetLocale()

	// path.Join ensures the resulting path uses slashes, suitable for fs.FS and i18n keys.
	resourcePath := path.Join(l.Dir, templatePath)

	// println("load replacer:", resourcePath)

	replacer := i18n.Replacer(locale, resourcePath)
	if replacer == nil {
		// Fallback to unlocalized content if no replacer is found for the locale/path.
		return assets.FS.Open(resourcePath)
	}

	content, err := fs.ReadFile(assets.FS, resourcePath)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(strings.NewReader(replacer.Replace(string(content)))), nil
}

// newLocalizedFSLoader creates a new LocalizedFSLoader with the specified base directory.
//
// dir is expected to be a slash-separated path.
func newLocalizedFSLoader(dir string) *LocalizedFSLoader {
	return &LocalizedFSLoader{
		Dir: dir,
	}
}
