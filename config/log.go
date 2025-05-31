// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
This file handles printing of server configuration.

Reflection is used to generically handle nested structures without hardcoding field names.
*/
package config

import (
	"fmt"
	"log"
	"net/url"
	"reflect"
	"slices"
)

// skippedFields contains field paths that should not be logged.
//
// These are specified using full struct path notation (e.g. "Parent.Child.Field").
var skippedFields = []string{
	"Basic.Token",
	"ContentProxies.RawImage",
	"ContentProxies.RawStatic",
	"ContentProxies.RawUgoira",
	"Limiter.PingHMAC",
	"Limiter.TurnstileSecretKey",
}

// printConfiguration prints the server configuration.
func (cfg *ServerConfig) printConfiguration() {
	v := reflect.ValueOf(cfg).Elem()

	log.Println("=============")
	log.Println("Configuration")
	log.Println("=============")
	cfg.logStruct("", "", "", v)
	log.Println("=============")
}

// logStruct recursively logs configuration structures.
//
// Parameters:
// - name: Field name.
// - fullPath: Full struct path.
// - indent: Current indentation level.
// - v: Reflection value of the current struct being processed.
func (cfg *ServerConfig) logStruct(name, fullPath, indent string, value reflect.Value) {
	valueType := value.Type()

	// Log the displayPath as a header if it's not empty
	if name != "" {
		log.Printf("%s%s:", indent, name)
		// Increase indent for child fields
		indent += "  "
	}

	for fieldIndex := range valueType.NumField() {
		field := valueType.Field(fieldIndex)
		if field.PkgPath != "" { // Skip unexported fields
			continue
		}

		// Build new display and full paths
		display := field.Name
		full := field.Name

		if fullPath != "" {
			full = fmt.Sprintf("%s.%s", fullPath, field.Name)
		}

		if shouldSkipField(full) {
			continue
		}

		fieldValue := value.Field(fieldIndex)
		cfg.logField(display, full, indent, fieldValue)
	}
}

// logField handles logging for an individual field value.
func (cfg *ServerConfig) logField(name, fullPath, indent string, value reflect.Value) {
	// Skip pointer fields entirely
	if value.Kind() == reflect.Ptr {
		return
	}

	// Print values of type url.URL using their String() representation
	if value.Kind() == reflect.Struct && value.Type() == reflect.TypeOf(url.URL{}) {
		url, ok := value.Interface().(url.URL)
		if !ok {
			log.Printf("%sError: Failed to convert %s to URL type", indent, name)

			return
		}

		log.Printf("%s%s: %s", indent, name, url.String())

		return
	}

	// Recursively handle nested structs
	if value.Kind() == reflect.Struct {
		cfg.logStruct(name, fullPath, indent, value)

		return
	}

	// Default case for simple value types
	log.Printf("%s%s: %v", indent, name, value.Interface())
}

// shouldSkipField determines whether a field should be skipped.
//
// Uses exact matching for fieldPath.
func shouldSkipField(fieldPath string) bool {
	return slices.Contains(skippedFields, fieldPath)
}
