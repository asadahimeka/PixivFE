// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
This file provides a tool that reads JSON input from stdin,
processes it to create a translation map using SuccintID for keys,
and outputs the result as JSON to stdout.

Used to convert translation data into a format suitable for use
by the i18n package.
*/
package main

import (
	"encoding/json"
	"os"

	"codeberg.org/pixivfe/pixivfe/i18n"
	"github.com/soluble-ai/go-jnode"
)

func main() {
	root := &jnode.Node{}

	err := json.NewDecoder(os.Stdin).Decode(root)
	if err != nil {
		panic(err)
	}

	translationMap := make(map[string]string)

	for i := range root.Size() {
		o := root.Get(i)
		msg := o.ToMap()["msg"].(string)
		file := o.ToMap()["file"].(string)
		translationMap[i18n.SuccintID(file, msg)] = msg
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	encoder.Encode(translationMap)
}
