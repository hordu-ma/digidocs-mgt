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
// The following carry intentionally large bodies and are left untouched:
//   - multipart uploads — document/data-asset handlers enforce their own limit
//     via r.ParseMultipartForm;
//   - git smart-HTTP endpoints (/git/) — push packs stream through git-backend;
//   - internal worker endpoints (/internal/) — result callbacks carry full
//     document text extraction / AI output, which routinely exceeds 1 MB.
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
	// Git smart-HTTP (/git/) and worker callbacks (/internal/) are namespaced
	// path prefixes that legitimately carry large bodies.
	return strings.Contains(r.URL.Path, "/git/") ||
		strings.Contains(r.URL.Path, "/internal/")
}
