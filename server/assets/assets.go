// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

/*
Package assets provides access to the application's embedded static assets.
*/
package assets

import (
	"io/fs"
)

// FS provides access to the embedded file system.
var FS fs.FS
