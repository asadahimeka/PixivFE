// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

// Based on https://gist.github.com/ZipFile/c9ebedb224406f4f11845ab700124362
// Don't panic on any errors in production btw.

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"codeberg.org/pixivfe/pixivfe/server/utils"
)

const (
	USER_AGENT     = "PixivAndroidApp/5.0.234 (Android 11; Pixel 5)"
	REDIRECT_URI   = "https://app-api.pixiv.net/web/v1/users/auth/pixiv/callback"
	LOGIN_URL      = "https://app-api.pixiv.net/web/v1/login"
	AUTH_TOKEN_URL = "https://oauth.secure.pixiv.net/auth/token"
	CLIENT_ID      = "MOBrBDS8blbauoSck0ZfDbtuzpyT"
	CLIENT_SECRET  = "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj"
)

type AppAPICredentials struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomStringURLSafe returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func generateRandomStringURLSafe(n int) (string, error) {
	b, err := generateRandomBytes(n)
	return base64.URLEncoding.EncodeToString(b), err
}

func s256(data []byte) string {
	// SHA-256
	hasher := sha256.New()
	hasher.Write(data)
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func oauth_pkce() (string, string) {
	code_verifier, err := generateRandomStringURLSafe(32)
	if err != nil {
		panic(err)
	}
	code_verifier = strings.Trim(code_verifier, "=")
	code_challenge := strings.Trim(s256([]byte(code_verifier)), "=")

	return code_verifier, code_challenge
}

func AppAPIRefresh(r *http.Request, refresh_token string) AppAPICredentials {
	var credentials AppAPICredentials
	body := []byte(fmt.Sprintf(`client_id=%s&client_secret=%s&grant_type=refresh_token&include_policy=true&refresh_token=%s`, CLIENT_ID, CLIENT_SECRET, refresh_token))
	req, err := http.NewRequestWithContext(r.Context(), "POST", AUTH_TOKEN_URL, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := utils.HTTPClient.Do(req)
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &credentials)
	if err != nil {
		panic(err)
	}

	return credentials
}

func AppAPILogin(r *http.Request) AppAPICredentials {
	var credentials AppAPICredentials
	code_verifier, code_challenge := oauth_pkce()

	// Let users open this URL and log in. Then enter the code.
	fmt.Printf(`%s?code_challenge=%s&code_challenge_method=S256&client=pixiv-android`, LOGIN_URL, code_challenge)
	var s string
	// Change this.
	fmt.Scanln(&s)

	body := []byte(fmt.Sprintf(`client_id=%s&client_secret=%s&code=%s&code_verifier=%s&grant_type=authorization_code&include_policy=true&redirect_uri=%s`, CLIENT_ID, CLIENT_SECRET, s, code_verifier, REDIRECT_URI))
	req, err := http.NewRequestWithContext(r.Context(), "POST", AUTH_TOKEN_URL, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := utils.HTTPClient.Do(req)
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &credentials)
	if err != nil {
		panic(err)
	}

	return credentials
}
