package routes

import (
	"net/http"

	"codeberg.org/pixivfe/pixivfe/assets/components/layout"
	"codeberg.org/pixivfe/pixivfe/assets/components/pages"
	"codeberg.org/pixivfe/pixivfe/server/requestcontext"
	"github.com/a-h/templ"
)

// BlockPage writes a block page using the provided reason.
func BlockPage(w http.ResponseWriter, r *http.Request, reason string) {
	w.Header().Set("Cache-Control", "no-store")

	pageData := pages.BlockData{
		Title:      "Blocked",
		Reason:     reason,
		StatusCode: requestcontext.FromRequest(r).StatusCode,
		Path:       r.URL.Path,
	}

	pageContent := pages.Block(pageData)

	ctxWithChildren := templ.WithChildren(r.Context(), pageContent)

	layout.Default(requestcontext.FromRequest(r).CommonData, pageData.Title, true).Render(ctxWithChildren, w)

	return
}
