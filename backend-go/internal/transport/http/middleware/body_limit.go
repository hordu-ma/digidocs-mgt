package middleware

import (
	"net/http"
	"strings"
)

// MaxJSONBodySize bounds the size of regular (non-upload) request bodies.
// JSON endpoints never legitimately send more than this, so capping it stops
// an oversized payload from being buffered into memory.
const MaxJSONBodySize = 1 << 20 // 1 MB

// LimitBody wraps eligible request bodies with http.MaxBytesReader so an
// oversized payload is rejected with 413 instead of being read into memory.
//
// Multipart uploads and the git smart-HTTP endpoints are left untouched: those
// carry intentionally large bodies and enforce (or don't need) their own limits
// — document/data-asset handlers via r.ParseMultipartForm, git via the backend.
func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && !bodyExempt(r) {
			r.Body = http.MaxBytesReader(w, r.Body, MaxJSONBodySize)
		}
		next.ServeHTTP(w, r)
	})
}

func bodyExempt(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/") {
		return true
	}
	if strings.HasPrefix(contentType, "application/x-git-") {
		return true
	}
	// Git smart-HTTP requests are namespaced under /git/.
	return strings.Contains(r.URL.Path, "/git/")
}
