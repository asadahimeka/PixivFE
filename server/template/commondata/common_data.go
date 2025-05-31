package commondata

import (
	"net/http"
	"net/url"

	"codeberg.org/pixivfe/pixivfe/audit"
	"codeberg.org/pixivfe/pixivfe/config"
	"codeberg.org/pixivfe/pixivfe/server/session"
	"codeberg.org/pixivfe/pixivfe/server/utils"
)

// PageCommonData holds common variables accessible in templates and handlers.
//
// It is automatically populated for each request and attached to the
// requestcontext.RequestContext.
//
// Usage:
//
//	// In an HTTP handler:
//	rc := requestcontext.FromRequest(r)
//	cd := rc.CommonData
//	// Now you can access fields like cd.BaseURL, cd.LoggedIn, etc.
type PageCommonData struct {
	BaseURL                  string
	CurrentPath              string
	CurrentPathWithParams    string
	HtmxCurrentPath          string
	FullURL                  string
	LoggedIn                 bool
	Queries                  map[string]string
	CookieList               map[string]string
	CookieListOrdered        []struct{ K, V string }
	RepoURL                  string
	Revision                 string
	RevisionHash             string
	IsHtmxRequest            bool
	LimiterEnabled           bool
	LinkToken                string
	DetectionMethod          string
	LinkTokenDetectionMethod string
	TurnstileDetectionMethod string
	TurnstileSiteKey         string
}

type LinkTokenGenerator func(*http.Request) (string, error)

// PopulatePageCommonData fills the PageCommonData struct from the request.
func PopulatePageCommonData(r *http.Request, data *PageCommonData, generateLinkToken LinkTokenGenerator) {
	data.BaseURL = utils.Origin(r)
	data.CurrentPath = r.URL.Path
	data.CurrentPathWithParams = r.URL.RequestURI()
	data.FullURL = r.URL.Scheme + "://" + r.Host + r.URL.Path

	// Parse HX-Current-URL to get proper current path for async requests.
	if htmxCurrentURL := r.Header.Get("HX-Current-URL"); htmxCurrentURL != "" {
		if parsedURL, err := url.Parse(htmxCurrentURL); err == nil {
			data.HtmxCurrentPath = parsedURL.Path
		}
	}

	data.LoggedIn = session.GetUserToken(r) != ""

	data.Queries = make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			data.Queries[k] = v[0]
		}
	}

	data.CookieList = make(map[string]string, len(session.AllCookieNames))
	data.CookieListOrdered = make([]struct{ K, V string }, 0, len(session.AllCookieNames))
	for _, name := range session.AllCookieNames {
		val := session.GetCookie(r, name)
		data.CookieList[string(name)] = val
		data.CookieListOrdered = append(data.CookieListOrdered, struct{ K, V string }{K: string(name), V: val})
	}

	data.RepoURL = config.GlobalConfig.Instance.RepoURL
	data.Revision = config.GlobalConfig.Instance.Revision
	data.RevisionHash = config.GlobalConfig.Instance.RevisionHash
	data.IsHtmxRequest = r.Header.Get("HX-Request") == "true"

	data.LimiterEnabled = config.GlobalConfig.Limiter.Enabled
	if data.LimiterEnabled && generateLinkToken != nil {
		var err error

		data.LinkToken, err = generateLinkToken(r)
		if err != nil {
			audit.GlobalAuditor.Logger.Errorln("Failed to generate link token: %v", err)

			data.LinkToken = ""
		}
	}

	data.DetectionMethod = config.GlobalConfig.Limiter.DetectionMethod
	data.LinkTokenDetectionMethod = config.LinkTokenDetectionMethod
	data.TurnstileDetectionMethod = config.TurnstileDetectionMethod
	data.TurnstileSiteKey = config.GlobalConfig.Limiter.TurnstileSitekey
}
