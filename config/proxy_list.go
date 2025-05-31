// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package config

const (
	BuiltInImageProxyPath  = "/proxy/i.pximg.net" // built-in proxy route for i.pximg.net
	BuiltInStaticProxyPath = "/proxy/s.pximg.net" // built-in proxy route for s.pximg.net
	BuiltInUgoiraProxyPath = "/proxy/ugoira.com"  // built-in proxy route for ugoira.com
)

// the list of proxies on /settings.
var BuiltInImageProxyList = []string{
	// !!!! WE ARE NOT AFFILIATED WITH MOST OF THE PROXIES !!!!
	"https://pixiv.ducks.party",
	"https://pximg.cocomi.eu.org",
	"https://i.suimoe.com",
	"https://i.yuki.sh",
	"https://pximg.obfs.dev",
	"https://pixiv.darkness.services",
	"https://pixiv.tatakai.top",
	"https://pi.169889.xyz",
	"https://i.pixiv.re",
	// "https://pximg.exozy.me", Rest in peace...
	// "https://mima.localghost.org/proxy/pximg", // only supports HTTP/0.9 and HTTP/1.x when using TLS.
	// "https://pximg.chaotic.ninja", // incompatible

	// VnPower: Please comment non-working sites instead of deleting them.
}
