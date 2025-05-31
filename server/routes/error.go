package routes

import (
	"net/http"

	"codeberg.org/pixivfe/pixivfe/assets/components/layout"
	"codeberg.org/pixivfe/pixivfe/assets/components/pages"
	"codeberg.org/pixivfe/pixivfe/server/requestcontext"
	"github.com/a-h/templ"
)

// ErrorPage writes an error page.
func ErrorPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	pageData := pages.ErrorData{
		Title:      "Error",
		Error:      requestcontext.FromRequest(r).RequestError,
		StatusCode: requestcontext.FromRequest(r).StatusCode,
	}

	pageContent := pages.Error(pageData)

	ctxWithChildren := templ.WithChildren(r.Context(), pageContent)

	layout.Default(requestcontext.FromRequest(r).CommonData, pageData.Title, true).Render(ctxWithChildren, w)
}
