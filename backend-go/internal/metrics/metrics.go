// Package metrics exposes a minimal Prometheus text-format endpoint without a
// client library dependency: HTTP request counts and latency per route, plus
// business gauges sampled from the database at scrape time.
package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type routeKey struct {
	route       string
	statusClass string
}

type routeStats struct {
	count       uint64
	durationSum float64
}

// Registry accumulates request metrics in process memory.
type Registry struct {
	mu     sync.Mutex
	routes map[routeKey]*routeStats
	start  time.Time
}

func NewRegistry() *Registry {
	return &Registry{routes: map[routeKey]*routeStats{}, start: time.Now()}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach Flush on the real writer
// (git smart-HTTP streams through here).
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Middleware records each request under its matched mux pattern (e.g.
// "GET /api/v1/documents/{documentID}"), never the raw path, so label
// cardinality stays bounded no matter how many ids are requested.
func (reg *Registry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		key := routeKey{route: route, statusClass: fmt.Sprintf("%dxx", sw.status/100)}
		reg.mu.Lock()
		st := reg.routes[key]
		if st == nil {
			st = &routeStats{}
			reg.routes[key] = st
		}
		st.count++
		st.durationSum += time.Since(start).Seconds()
		reg.mu.Unlock()
	})
}

// gauge is one business metric sampled at scrape time.
type gauge struct {
	name, help, query string
	labeled           bool // query returns (label, value) rows
	label             string
}

var gauges = []gauge{
	{name: "digidocs_assistant_requests", help: "AI tasks by status.", query: `SELECT status, COUNT(*) FROM assistant_requests GROUP BY status`, labeled: true, label: "status"},
	{name: "digidocs_assistant_requests_oldest_pending_seconds", help: "Age of the oldest pending AI task (queue backlog).", query: `SELECT COALESCE(EXTRACT(EPOCH FROM NOW() - MIN(created_at)), 0) FROM assistant_requests WHERE status = 'pending'`},
	{name: "digidocs_ai_tokens_24h", help: "Model tokens used by completed AI tasks in the last 24 hours.", query: `SELECT COALESCE(SUM((output->'usage'->>'total_tokens')::bigint), 0) FROM assistant_requests WHERE completed_at > NOW() - INTERVAL '24 hours' AND output->'usage'->>'total_tokens' ~ '^[0-9]+$'`},
	{name: "digidocs_documents", help: "Documents by status (not deleted).", query: `SELECT current_status::text, COUNT(*) FROM documents WHERE is_deleted = false GROUP BY 1`, labeled: true, label: "status"},
	{name: "digidocs_handovers_open", help: "Handovers not yet completed or cancelled.", query: `SELECT COUNT(*) FROM graduation_handovers WHERE status IN ('generated', 'pending_confirm')`},
	{name: "digidocs_notifications_unread", help: "Unread in-app notifications.", query: `SELECT COUNT(*) FROM notifications WHERE read_at IS NULL`},
	{name: "digidocs_text_chunks", help: "Indexed document text chunks.", query: `SELECT COUNT(*) FROM document_text_chunks`},
}

// Handler serves the text exposition format. With a non-empty token it
// requires "Authorization: Bearer <token>"; with an empty token the endpoint
// is disabled (404) so it is never exposed by accident.
func (reg *Registry) Handler(db *sql.DB, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		reg.write(w)
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			writeGauges(ctx, w, db)
		}
	})
}

func (reg *Registry) write(w io.Writer) {
	reg.mu.Lock()
	keys := make([]routeKey, 0, len(reg.routes))
	for k := range reg.routes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		return keys[i].statusClass < keys[j].statusClass
	})
	fmt.Fprintln(w, "# HELP digidocs_http_requests_total HTTP requests by route and status class.")
	fmt.Fprintln(w, "# TYPE digidocs_http_requests_total counter")
	for _, k := range keys {
		fmt.Fprintf(w, "digidocs_http_requests_total{route=%q,status=%q} %d\n", k.route, k.statusClass, reg.routes[k].count)
	}
	fmt.Fprintln(w, "# HELP digidocs_http_request_duration_seconds_sum Total time spent serving requests.")
	fmt.Fprintln(w, "# TYPE digidocs_http_request_duration_seconds_sum counter")
	for _, k := range keys {
		fmt.Fprintf(w, "digidocs_http_request_duration_seconds_sum{route=%q,status=%q} %.6f\n", k.route, k.statusClass, reg.routes[k].durationSum)
	}
	reg.mu.Unlock()
	fmt.Fprintln(w, "# HELP digidocs_uptime_seconds Seconds since the API process started.")
	fmt.Fprintln(w, "# TYPE digidocs_uptime_seconds gauge")
	fmt.Fprintf(w, "digidocs_uptime_seconds %.0f\n", time.Since(reg.start).Seconds())
}

func writeGauges(ctx context.Context, w io.Writer, db *sql.DB) {
	for _, g := range gauges {
		rows, err := db.QueryContext(ctx, g.query)
		if err != nil {
			// A missing table during a partial rollback must not break the scrape.
			fmt.Fprintf(w, "# %s unavailable: %s\n", g.name, strings.ReplaceAll(err.Error(), "\n", " "))
			continue
		}
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n", g.name, g.help, g.name)
		for rows.Next() {
			if g.labeled {
				var label string
				var value float64
				if rows.Scan(&label, &value) == nil {
					fmt.Fprintf(w, "%s{%s=%q} %g\n", g.name, g.label, label, value)
				}
			} else {
				var value float64
				if rows.Scan(&value) == nil {
					fmt.Fprintf(w, "%s %g\n", g.name, value)
				}
			}
		}
		rows.Close()
	}
}
