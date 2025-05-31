// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package template

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"codeberg.org/pixivfe/pixivfe/server/requestcontext"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/utils"
	"github.com/CloudyKit/jet/v6"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

var views *jet.Set // global variable, yes.

// Setup initializes the template engine.
func Setup(disableCache bool) {
	loader := newLocalizedFSLoader("assets/views")
	if disableCache {
		views = jet.NewSet(
			loader,
			jet.InDevelopmentMode(), // disable cache
		)
	} else {
		views = jet.NewSet(
			loader,
		)
	}

	for fnName, fn := range getTemplateFunctions() {
		views.AddGlobal(fnName, fn)
	}
}

func RenderHTML[T any](w http.ResponseWriter, r *http.Request, data T) error {
	return RenderWithContentType(w, r, "text/html; charset=utf-8", data)
}

func RenderWithContentType[T any](w http.ResponseWriter, r *http.Request, contentType string, data T) error {
	// Render and get ETag
	content, etag, serverTiming, err := Render(getTemplatingVariables(r), data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", contentType)

	// Set strong ETag
	// NOTE: the ngx_brotli nginx module appears to weaken strong ETags (even though gzipping is fine)
	w.Header().Set("ETag", `"`+etag+`"`)

	// This is a conservative fallback; the proper cache-control
	// directives should be set in the route handler
	if w.Header().Get("Cache-Control") == "" {
		if session.GetUserToken(r) != "" {
			w.Header().Set("Cache-Control", "no-store")
		} else {
			w.Header().Set("Cache-Control", "private, max-age=60")
		}
	}

	// Write Server-Timing header
	w.Header().Add("Server-Timing", serverTiming)

	// Set Vary: Cookie to prevent user preferences from affecting the shared cache
	//
	// This negatively affects HTTP cache hit rate, but we don't have the option of
	// client-side hydration via JS nor ESI so these will have to do
	w.Header().Set("Vary", "Cookie")

	w.WriteHeader(requestcontext.FromRequest(r).StatusCode)

	_, err = w.Write(content)

	return err
}

func Render[T any](variables jet.VarMap, data T) ([]byte, string, string, error) {
	templateName, found := strings.CutPrefix(reflect.TypeFor[T]().Name(), "Data_")
	if !found {
		log.Panicf("struct name does not start with 'Data_': %s", templateName)
	}

	// Determine template path based on if it's a partial
	templatePath := templateName + ".jet.html"
	if strings.HasSuffix(templateName, "Partial") {
		templatePath = "partials/" + templatePath
	}

	template, err := views.GetTemplate(templatePath)
	if err != nil {
		return nil, "", "", err
	}

	template, err = views.Parse(templatePath, template.String())
	if err != nil {
		return nil, "", "", err
	}

	buf := &bytes.Buffer{}

	renderStart := time.Now()
	// Create minifier
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	minWriter := m.Writer("text/html", buf)

	// Execute template directly to minifier writer
	if err := template.Execute(minWriter, variables, data); err != nil {
		return nil, "", "", err
	}

	renderDuration := time.Since(renderStart).Milliseconds()

	minifyStart := time.Now()
	// Close the minifier writer to flush any remaining content
	if err := minWriter.Close(); err != nil {
		return nil, "", "", err
	}

	minifyDuration := time.Since(minifyStart).Milliseconds()

	// Get the final bytes
	minifiedBytes := buf.Bytes()
	etag := utils.GenerateETag(minifiedBytes)

	// Create Server-Timing value
	serverTiming := fmt.Sprintf(
		"template-render;dur=%d;desc=\"Template render\", html-minify;dur=%d;desc=\"HTML minify\"",
		renderDuration,
		minifyDuration,
	)

	return minifiedBytes, etag, serverTiming, nil
}

// getTemplatingVariables extracts common data from the request context
// and builds a jet.VarMap for template rendering.
func getTemplatingVariables(r *http.Request) jet.VarMap {
	cd := requestcontext.FromRequest(r).CommonData

	// We want to iterate over the struct’s fields
	rv := reflect.ValueOf(cd)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	rt := rv.Type()
	vars := jet.VarMap{}

	for i := range rt.NumField() {
		sf := rt.Field(i)
		// Only exported fields are addressable and Interface-able
		fv := rv.Field(i)
		if !fv.CanInterface() {
			continue
		}

		// Use the struct field name as the template variable name
		vars.Set(sf.Name, fv.Interface())
	}

	return vars
}
