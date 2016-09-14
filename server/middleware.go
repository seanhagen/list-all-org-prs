package server

import (
	"net/http"
	"strings"
)

func getProperPath(r *http.Request, s *Server) string {
	path := r.URL.Path

	_, params, _ := s.router.Lookup(r.Method, r.URL.Path)
	for _, p := range params {
		r := ":" + p.Key
		path = strings.Replace(path, p.Value, r, -1)
	}
	return path
}

func tokenAuth(s *Server) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := getProperPath(r, s)

			var route Route
			method := s.Routes[r.Method]
			if method != nil {
				route = s.Routes[r.Method][path]
			}

			switch route.AuthType {
			case AUTHTOKEN:
				authToken(h, w, r)
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}

func authToken(h http.Handler, w http.ResponseWriter, r *http.Request) {
	h.ServeHTTP(w, r)
}
