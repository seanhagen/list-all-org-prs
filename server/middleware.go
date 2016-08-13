package server

import (
	"net/http"
)

func tokenAuth(routes RouteMap) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := routes[r.Method][r.URL.Path]

			switch route.AuthType {
			case AUTHTOKEN:
				authToken(h, w, r)
			}
		})
	}
}

func authToken(h http.Handler, w http.ResponseWriter, r *http.Request) {
	h.ServeHTTP(w, r)
}
