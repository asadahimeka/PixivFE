// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
This file contains global functions and variables
for setting and retrieving the current locale, as well as
functions for translating user-facing errors and formatted strings.
*/
package i18n

import (
	"errors"
	"fmt"

	"github.com/timandy/routine"
)

var goroutineLocale = routine.NewInheritableThreadLocal[string]()

const BaseLocale = "en"

func GetLocale() string {
	locale := goroutineLocale.Get()
	if locale == "" {
		locale = BaseLocale
	}

	return locale
}

func SetLocale(locale string) {
	goroutineLocale.Set(locale)
}

func Error(text string) error {
	text = lookupSkipStack2(GetLocale(), text)

	return errors.New(text) //nolint:err113 // Intentionally creates dynamic errors.
}

func Errorf(format string, a ...any) error {
	format = lookupSkipStack2(GetLocale(), format)

	return fmt.Errorf(format, a...) //nolint:err113 // Intentionally creates dynamic errors.
}

func Sprintf(format string, a ...any) string {
	format = lookupSkipStack2(GetLocale(), format)

	return fmt.Sprintf(format, a...)
}

// Tr returns a translation for the provided string.
func Tr(text string) string {
	return lookupSkipStack2(GetLocale(), text)
}
