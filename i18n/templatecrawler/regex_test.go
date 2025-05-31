// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
This file provides tests for the regular expressions used in the i18n crawler.
*/
package main

import "testing"

func TestRegex(t *testing.T) {
	if !re_command_fullmatch.MatchString(`{{- extends "layout/default" }}
{{- block body() }}`) {
		t.Error("re_command is broken")
	}

	if stripJetComments("awawa {* awawawawawa*} awawawa") != `awawa  awawawa` {
		t.Error("re_comment is broken")
	}
}
