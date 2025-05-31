// Copyright 2023 - 2025, VnPower and the PixivFE contributors
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"fmt"
	"net/http"

	"codeberg.org/pixivfe/pixivfe/core/requests"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"github.com/goccy/go-json"
)

type PixivSettingsResponse struct {
	Languages struct {
		Ja   string `json:"ja"`
		En   string `json:"en"`
		Ko   string `json:"ko"`
		Zh   string `json:"zh"`
		ZhTw string `json:"zh_tw"`
	} `json:"languages"`
	MailAddressChangeLink                 string `json:"mailAddressChangeLink"`
	PixivIDLink                           string `json:"pixivIdLink"`
	PasswordChangeLink                    string `json:"passwordChangeLink"`
	SecurityHistoryLink                   string `json:"securityHistoryLink"`
	ShouldShowTwoFactorAuthenticationLink bool   `json:"shouldShowTwoFactorAuthenticationLink"`
	IsTwoFactorAuthenticationEnabled      bool   `json:"isTwoFactorAuthenticationEnabled"`
	TwoFactorAuthenticationLink           string `json:"twoFactorAuthenticationLink"`
	SocialLoginLink                       string `json:"socialLoginLink"`
	MessageFlag                           int    `json:"messageFlag"`
	HideAiWorks                           bool   `json:"hideAiWorks"`
	ReadingStatusOptout                   bool   `json:"readingStatusOptout"`
	Location                              string `json:"location"`
	PasskeyCount                          int    `json:"passkeyCount"`
	PasskeysLink                          string `json:"passkeysLink"`
	NoPassword                            bool   `json:"noPassword"`
	NoEmailAuthorized                     bool   `json:"noEmailAuthorized"`
	OptoutRenewal                         bool   `json:"optoutRenewal"`
}

type SettingsSelfResponse struct {
	UserStatus struct {
		UserID          string `json:"user_id"`
		UserStatus      string `json:"user_status"`
		UserAccount     string `json:"user_account"`
		UserName        string `json:"user_name"`
		UserPremium     string `json:"user_premium"`
		UserBirth       string `json:"user_birth"`
		UserXRestrict   string `json:"user_x_restrict"`
		UserCreateTime  string `json:"user_create_time"`
		UserMailAddress string `json:"user_mail_address"`
		ProfileImg      struct {
			Main string `json:"main"`
		} `json:"profile_img"`
		Age         int  `json:"age"`
		IsLoggedIn  bool `json:"is_logged_in"`
		StampSeries []struct {
			Slug   string `json:"slug"`
			Name   string `json:"name"`
			Stamps []int  `json:"stamps"`
		} `json:"stamp_series"`
		EmojiSeries []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"emoji_series"`
		AdsDisabled          bool   `json:"ads_disabled"`
		ShowAds              bool   `json:"show_ads"`
		TwitterAccount       bool   `json:"twitter_account"`
		IsIllustCreator      bool   `json:"is_illust_creator"`
		IsNovelCreator       bool   `json:"is_novel_creator"`
		HideAiWorks          bool   `json:"hide_ai_works"`
		ReadingStatusEnabled bool   `json:"reading_status_enabled"`
		IllustMaskRules      []any  `json:"illust_mask_rules"`
		Location             string `json:"location"`
		SensitiveViewSetting int    `json:"sensitive_view_setting"`
	} `json:"user_status"`
}

func GetPixivSettings(r *http.Request) (*PixivSettingsResponse, error) {
	url := GetPixivSettingsURL()

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	resp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	resp = RewriteContentURLs(r, resp)

	var settingsResult PixivSettingsResponse
	if err := json.Unmarshal(resp, &settingsResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Pixiv settings: %w", err)
	}

	return &settingsResult, nil
}

func GetSettingsSelf(r *http.Request) (*SettingsSelfResponse, error) {
	url := GetSettingsSelfURL()

	cookies := map[string]string{
		"PHPSESSID": session.GetUserToken(r),
	}

	resp, err := requests.FetchJSONBodyField(r.Context(), url, cookies, r.Header)
	if err != nil {
		return nil, err
	}

	resp = RewriteContentURLs(r, resp)

	var settingsResult SettingsSelfResponse
	if err := json.Unmarshal(resp, &settingsResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal personal settings: %w", err)
	}

	return &settingsResult, nil
}
