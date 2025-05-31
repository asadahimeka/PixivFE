// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package utils

import (
	"crypto/tls"
	"net/http"
)

const (
	// ClientSessionCacheSize defines the size of the TLS session cache.
	ClientSessionCacheSize = 20

	// MaxIdleConnsPerHost defines maximum idle connections to keep per host.
	MaxIdleConnsPerHost = 20

	// BufferSize defines the read and write buffer size in bytes (32KB).
	BufferSize = 32 * 1024
)

// HTTPClient is a pre-configured http.Client.
//
// It serves as a base HTTP client used across different packages.
var HTTPClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			ClientSessionCache: tls.NewLRUClientSessionCache(ClientSessionCacheSize),
			MinVersion:         tls.VersionTLS12,
		},
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        0,
		MaxIdleConnsPerHost: MaxIdleConnsPerHost,
		WriteBufferSize:     BufferSize,
		ReadBufferSize:      BufferSize,
	},
}

// // HTTP3Client is a pre-configured http.Client with HTTP/3 capabilities.
// var HTTP3Client = &http.Client{
// 	Transport: &http3.Transport{
// 		QUICConfig: &quic.Config{
// 		},
// 	},
// }
