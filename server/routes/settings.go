// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/core"
	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/i18n"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/template"
	"codeberg.org/pixivfe/pixivfe/server/utils"
	"github.com/goccy/go-json"
)

var (
	r_csrf = regexp.MustCompile(`\\"token\\":\\"([0-9a-f]+)\\"`)
	r_p_ab = regexp.MustCompile(`\\"pAbDId\\":(\d+)`)
)

func setToken(w http.ResponseWriter, r *http.Request) (string, error) {
	token := r.FormValue("token")
	if token == "" {
		return "", i18n.Error("You submitted an empty/invalid form.")
	}

	cookies := map[string]string{
		"PHPSESSID": token,
	}

	// Request API route to check if token is valid
	url := core.GetNewestFromFollowingURL("illust", "all", "1")
	_, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return "", i18n.Error("Cannot authorize with supplied token. (API returned not OK)")
	}

	// Request artwork page to extract csrf token from body
	// THE TEST URL IS NSFW!
	rawResp, err := requests.PerformGET(r.Context(), "https://www.pixiv.net/en/artworks/115365120", cookies, r.Header)
	if err != nil {
		return "", err
	}

	if rawResp.StatusCode != http.StatusOK {
		return "", i18n.Error("Cannot authorize with supplied token. (Page returned not OK)")
	}

	// Extract CSRF token
	csrfMatches := r_csrf.FindStringSubmatch(string(rawResp.Body))
	if len(csrfMatches) < 2 {
		return "", i18n.Error("Unable to extract CSRF token from response.")
	}
	csrf := csrfMatches[1]

	// Extract personal ID
	personalIDMatches := r_p_ab.FindStringSubmatch(string(rawResp.Body))
	if len(personalIDMatches) < 2 {
		return "", i18n.Error("Unable to extract a personal ID from response.")
	}
	personalID := personalIDMatches[1]

	if personalID == "" {
		return "", i18n.Error("Cannot authorize with supplied token. (Invalid personal ID)")
	}

	// Get user ID from token
	// The left part of the token is the member ID
	userID := strings.Split(token, "_")

	// Get user information
	userInfoURL := core.GetUserInformationURL(userID[0], "1")
	rawResp2, err := requests.FetchJSONBodyField(r.Context(), userInfoURL, cookies, r.Header)
	if err != nil {
		return "", err
	}

	rawResp2 = core.RewriteContentURLs(r, rawResp2)

	// Parse user info
	var userInfo struct {
		UserID   string `json:"userId"`
		Username string `json:"name"`
		Avatar   string `json:"imageBig"`
	}

	if err := json.Unmarshal(rawResp2, &userInfo); err != nil {
		return "", err
	}

	// Set all cookies
	session.SetCookie(w, r, session.Cookie_Token, token)
	session.SetCookie(w, r, session.Cookie_CSRF, csrf)
	session.SetCookie(w, r, session.Cookie_P_AB, personalID)
	session.SetCookie(w, r, session.Cookie_Username, userInfo.Username)
	session.SetCookie(w, r, session.Cookie_UserID, userInfo.UserID)
	session.SetCookie(w, r, session.Cookie_UserAvatar, userInfo.Avatar)

	return i18n.Sprintf("Successfully logged in."), nil
}

func setImageServer(w http.ResponseWriter, r *http.Request) (string, error) {
	customProxy := r.FormValue("custom-image-proxy")
	selectedProxy := r.FormValue("image-proxy")

	switch {
	case selectedProxy == "custom" && customProxy != "":
		return handleCustomProxy(w, r, customProxy)
	case selectedProxy != "":
		return handleSelectedProxy(w, r, selectedProxy)
	default:
		session.ClearCookie(w, r, session.Cookie_ImageProxy)
		return i18n.Sprintf("Image proxy server cleared. Using default proxy."), nil
	}
}

func handleCustomProxy(w http.ResponseWriter, r *http.Request, customProxy string) (string, error) {
	var proxyURL string

	if customProxy == config.BuiltInImageProxyPath {
		proxyURL = config.BuiltInImageProxyPath
	} else {
		parsedURL, err := utils.ValidateURL(customProxy, "Custom image proxy")
		if err != nil {
			return "", err
		}
		proxyURL = parsedURL.String()
	}

	session.SetCookie(w, r, session.Cookie_ImageProxy, proxyURL)
	return i18n.Sprintf("Image proxy server set successfully to: %s", proxyURL), nil
}

func handleSelectedProxy(w http.ResponseWriter, r *http.Request, selectedProxy string) (string, error) {
	proxyURL := selectedProxy

	session.SetCookie(w, r, session.Cookie_ImageProxy, proxyURL)
	return i18n.Sprintf("Image proxy server set successfully to: %s", proxyURL), nil
}

func setVisualEffects(w http.ResponseWriter, r *http.Request) (string, error) {
	visualEffects := r.FormValue("visual-effects")
	seasonalEffects := r.FormValue("seasonal-effects")

	isSuccessful := false

	if visualEffects == "on" {
		session.SetCookie(w, r, session.Cookie_VisualEffectsEnabled, "true")
		isSuccessful = true
	} else if visualEffects == "" {
		session.SetCookie(w, r, session.Cookie_VisualEffectsEnabled, "false")
		isSuccessful = true
	}

	if seasonalEffects == "on" {
		session.SetCookie(w, r, session.Cookie_SeasonalEffectsDisabled, "true")
		isSuccessful = true
	} else {
		session.SetCookie(w, r, session.Cookie_SeasonalEffectsDisabled, "false")
		isSuccessful = true
	}

	if isSuccessful {
		return i18n.Sprintf("Visual effects preference updated successfully."), nil
	}

	return "", i18n.Error("Invalid visual effects preference.")
}

func setTimeZone(w http.ResponseWriter, r *http.Request) (string, error) {
	timeZone := r.FormValue("time-zone")

	// Validate timezone
	_, err := session.ValidateTimezone(timeZone)
	if err != nil {
		return "", i18n.Error("Invalid timezone specified.")
	}

	if timeZone == "" {
		session.ClearCookie(w, r, session.Cookie_TZ)
		return i18n.Sprintf("Timezone reset to UTC."), nil
	}

	session.SetCookie(w, r, session.Cookie_TZ, timeZone)
	return i18n.Sprintf("Timezone updated successfully to %s.", timeZone), nil
}

func setNovelFontType(w http.ResponseWriter, r *http.Request) (string, error) {
	fontType := r.FormValue("font-type")
	if fontType != "" {
		session.SetCookie(w, r, session.Cookie_NovelFontType, fontType)
		return i18n.Sprintf("Novel font type updated successfully."), nil
	}

	return "", i18n.Error("Invalid font type.")
}

func setNovelViewMode(w http.ResponseWriter, r *http.Request) (string, error) {
	viewMode := r.FormValue("view-mode")
	if viewMode == "1" || viewMode == "2" || viewMode == "" {
		session.SetCookie(w, r, session.Cookie_NovelViewMode, viewMode)
		return i18n.Sprintf("Novel view mode updated successfully."), nil
	}

	return "", i18n.Error("Invalid view mode.")
}

func setThumbnailToNewTab(w http.ResponseWriter, r *http.Request) (string, error) {
	ttnt := r.FormValue("ttnt")
	if ttnt == "_blank" {
		session.SetCookie(w, r, session.Cookie_ThumbnailToNewTab, ttnt)
		return i18n.Sprintf("Thumbnails will now open in a new tab."), nil
	}

	session.SetCookie(w, r, session.Cookie_ThumbnailToNewTab, "_self")
	return i18n.Sprintf("Thumbnails will now open in the same tab."), nil
}

func setArtworkPreview(w http.ResponseWriter, r *http.Request) (string, error) {
	value := r.FormValue("app")
	if value == "cover" || value == "button" || value == "" {
		session.SetCookie(w, r, session.Cookie_ArtworkPreview, value)
		return i18n.Sprintf("Artwork preview setting updated successfully."), nil
	}

	return "", i18n.Error("Invalid artwork preview setting.")
}

func setFilter(w http.ResponseWriter, r *http.Request) (string, error) {
	visibilityArtR18 := r.FormValue("visibility-art-r18")
	visibilityArtR18G := r.FormValue("visibility-art-r18g")
	visibilityArtAI := r.FormValue("visibility-art-ai")

	session.SetCookie(w, r, session.Cookie_VisibilityArtR18, visibilityArtR18)
	session.SetCookie(w, r, session.Cookie_VisibilityArtR18G, visibilityArtR18G)
	session.SetCookie(w, r, session.Cookie_VisibilityArtAI, visibilityArtAI)

	return i18n.Sprintf("Filter settings updated successfully."), nil
}

func setLogout(w http.ResponseWriter, r *http.Request) (string, error) {
	// Clear-Site-Data header with wildcard to clear everything
	w.Header().Set("Clear-Site-Data", "*")

	// Cookie clearing as fallback
	session.ClearCookie(w, r, session.Cookie_Token)
	session.ClearCookie(w, r, session.Cookie_CSRF)
	session.ClearCookie(w, r, session.Cookie_P_AB)
	session.ClearCookie(w, r, session.Cookie_Username)
	session.ClearCookie(w, r, session.Cookie_UserID)
	session.ClearCookie(w, r, session.Cookie_UserAvatar)
	return i18n.Sprintf("Successfully logged out."), nil
}

func setCookie(w http.ResponseWriter, r *http.Request) (string, error) {
	key := r.FormValue("key")
	value := r.FormValue("value")

	for _, cookieName := range session.AllCookieNames {
		if string(cookieName) == key {
			session.SetCookie(w, r, cookieName, value)
			return i18n.Sprintf("Cookie %s set successfully.", key), nil
		}
	}
	return "", i18n.Errorf("Invalid Cookie Name: %s", key)
}

func clearCookie(w http.ResponseWriter, r *http.Request) (string, error) {
	key := r.FormValue("key")

	for _, cookieName := range session.AllCookieNames {
		if string(cookieName) == key {
			session.ClearCookie(w, r, cookieName)
			return i18n.Sprintf("Cookie %s cleared successfully.", key), nil
		}
	}

	return "", i18n.Errorf("Invalid Cookie Name: %s", key)
}

// setRawCookie processes a multi-line string of key=value pairs
// from the "raw" form value and sets corresponding valid session cookies.
//
// It returns a status message indicating success, including skipped count if any.
func setRawCookie(w http.ResponseWriter, r *http.Request) (string, error) {
	raw := r.FormValue("raw")

	lines := strings.Split(raw, "\n")

	var (
		appliedCount int // Tracks settings that were actually applied
		skippedCount int // Tracks skipped lines
	)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines or lines that are just comments,
		// but don't count them as skipped entries
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Split into exactly two parts: key and value
		parts := strings.SplitN(trimmedLine, "=", 2)
		if len(parts) != 2 {
			// Malformed lines are skipped and counted as such
			skippedCount++
			continue
		}

		// Trim potential whitespace around key and value
		keyStr := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Convert key name
		name := session.CookieName(keyStr)

		// Check if the cookie name is allowed
		if !slices.Contains(session.AllCookieNames, name) {
			skippedCount++
			continue
		}

		// Set the cookie
		session.SetCookie(w, r, name, value)
		appliedCount++
	}

	// Applied count is the base message
	msgApplied := ""
	if appliedCount > 0 {
		// Manually handle plurals since i18n.Sprintf doesn't
		if appliedCount == 1 {
			msgApplied = i18n.Sprintf("Applied 1 setting successfully")
		} else {
			msgApplied = i18n.Sprintf("Applied %d settings successfully", appliedCount)
		}
	}

	// Skipped count is an optional addition
	msgSkipped := ""
	if skippedCount > 0 {
		msgSkipped = i18n.Sprintf("Skipped %d invalid or unknown entries", skippedCount)
	}

	// Combine the parts conditionally
	switch {
	case appliedCount > 0 && skippedCount > 0:
		// Both applied and skipped: Combine with a period and space.
		return fmt.Sprintf("%s. %s.", msgApplied, msgSkipped), nil
	case appliedCount > 0:
		// Only applied
		return fmt.Sprintf("%s.", msgApplied), nil
	case skippedCount > 0:
		// Only skipped (implies appliedCount is 0)
		return fmt.Sprintf("No valid settings found. %s.", msgSkipped), nil
	default: // appliedCount == 0 && skippedCount == 0
		// Neither applied nor skipped
		return i18n.Sprintf("No valid settings found in the input."), nil
	}
}

func resetAll(w http.ResponseWriter, r *http.Request) (string, error) {
	// Clear-Site-Data header with wildcard to clear everything
	w.Header().Set("Clear-Site-Data", "*")

	// Cookie clearing as fallback
	session.ClearAllCookies(w, r)
	return i18n.Sprintf("All preferences have been reset to default values."), nil
}

func SettingsPage(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Cache-Control", "private, max-age=60")

	var profile core.SettingsSelfResponse

	if session.GetUserToken(r) != "" {
		p, err := core.GetSettingsSelf(r)
		if err != nil {
		}

		profile = *p
	}

	return template.RenderHTML(w, r, Data_settings{
		SelfSettings:       profile,
		ProxyList:          config.BuiltInImageProxyList,
		DefaultProxyServer: config.GlobalConfig.ContentProxies.Image.String(),
	})
}

func handleAJAXResponse(w http.ResponseWriter, message string, err error) {
	w.Header().Set("Content-Type", "text/html")

	var (
		html       string
		statusCode int
	)

	if err != nil {
		statusCode = http.StatusBadRequest
		html = i18n.Sprintf(
			`<div class="form-htmx-target" hidden></div><div id="form-htmx-response" class="flex items-center w-fit bg-yellow-500/10 border border-yellow-500 text-yellow-100 fill-yellow-100 text-sm rounded-lg gap-4 py-3 px-4 transition-opacity duration-300"> %s<button type="button" class="group size-fit cursor-pointer hover:bg-yellow-500/20 active:scale-95 transition rounded-full p-1 -me-1" aria-label="Close" hx-on:click="const el = this.closest('#form-htmx-response'); el.style.opacity = '0'; setTimeout(() => el.remove(), 200)"><span class="material-symbols-rounded-20>close</span></button></div>`, err.Error())
	} else {
		statusCode = http.StatusOK
		html = i18n.Sprintf(
			`<div class="form-htmx-target" hidden></div><div id="form-htmx-response" class="flex items-center w-fit bg-blue-500/10 border border-blue-500 text-blue-100 fill-blue-100 text-sm rounded-lg gap-4 py-3 px-4 transition-opacity duration-300"> %s<button type="button" class="group size-fit cursor-pointer hover:bg-blue-500/20 active:scale-95 transition rounded-full p-1 -me-1" aria-label="Close" hx-on:click="const el = this.closest('#form-htmx-response'); el.style.opacity = '0'; setTimeout(() => el.remove(), 200)"><span class="material-symbols-rounded-20>close</span></button></div>`, message)
	}

	w.WriteHeader(statusCode)

	if _, werr := w.Write([]byte(html)); werr != nil {
		audit.GlobalAuditor.Logger.Error("Error writing response: %v", werr)
	}
}

func SettingsPost(w http.ResponseWriter, r *http.Request) error {
	actionType := GetPathVar(r, "type") // TODO: this should just be called "action"

	// Map setting types to their respective functions.
	actions := map[string]func(http.ResponseWriter, *http.Request) (string, error){
		"imageServer":       setImageServer,
		"token":             setToken,
		"logout":            setLogout,
		"timeZone":          setTimeZone,
		"reset-all":         resetAll,
		"novelFontType":     setNovelFontType,
		"thumbnailToNewTab": setThumbnailToNewTab,
		"novelViewMode":     setNovelViewMode,
		"artworkPreview":    setArtworkPreview,
		"filter":            setFilter,
		"visualEffects":     setVisualEffects,
		"set-cookie":        setCookie,
		"clear-cookie":      clearCookie,
		"raw":               setRawCookie,
	}

	var (
		message string
		err     error
	)

	if action, ok := actions[actionType]; ok {
		message, err = action(w, r)
	} else {
		err = i18n.Error("No such setting is available.")
	}

	isHtmx := r.Header.Get("HX-Request") == "true"
	returnPath := r.FormValue("returnPath")

	if isHtmx {
		if returnPath != "" && err == nil {
			http.Redirect(w, r, returnPath, http.StatusSeeOther)
		} else {
			handleAJAXResponse(w, message, err)
		}
		return nil
	}

	if err != nil {
		return err
	}

	http.Redirect(w, r, returnPath, http.StatusSeeOther)
	// utils.RedirectToWhenceYouCame(w, r, returnPath)

	return nil
}
