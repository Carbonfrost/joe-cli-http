package httpserver

import (
	"net/http"
	"strings"
)

func newFileServerHandler(staticDir string, hideDirListing bool) http.Handler {
	result := http.FileServer(http.Dir(staticDir))
	if hideDirListing {
		result = hideListing(result)
	}
	return result
}

func hideListing(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/") {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		next.ServeHTTP(w, req)
	}
}
