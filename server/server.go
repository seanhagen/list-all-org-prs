package server

import (
	"github.com/julienschmidt/httprouter"
	// "net/http"
)

type AuthType int

const (
	AUTH_NONE  AuthType = 1
	AUTH_TOKEN AuthType = 2
)

type (
	HandleFunc func() httprouter.Handle

	Route struct {
		AuthType AuthType
		Handler  HandleFunc
	}

	RouteMap map[string]map[string]Route
)

// CreateRoute does a thing
func CreateRoute(auth AuthType, handler HandleFunc) Route {
	if auth != AUTH_TOKEN {
		auth = AUTH_NONE
	}

	return Route{
		AuthType: auth,
		Handler:  handler,
	}
}

func GetEmptyRoutes() RouteMap {
	routes := make(RouteMap)
	verbs := []string{"GET", "PUT", "POST", "OPTIONS", "PATCH", "DELETE"}
	for _, v := range verbs {
		routes[v] = make(map[string]Route)
	}
	return routes
}
