package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	wrapped := handler

	for idx := len(middlewares) - 1; idx >= 0; idx-- {
		wrapped = middlewares[idx](wrapped)
	}

	return wrapped
}
