package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/rs/cors"
	"net/http"
	"os"
)

// AuthType stores the auth type used to determine if a route needs to check the token
type AuthType int

const (
	// AUTHNONE means no checking the token
	AUTHNONE AuthType = 1
	// AUTHTOKEN means check for a valid JWT
	AUTHTOKEN AuthType = 2
)

type (
	// Route holds the AuthType used to authenticate requests, as well as the
	// handler used to actually handle the route
	Route struct {
		AuthType AuthType
		Handler  httprouter.Handle
	}

	// RouteMap contains the routes used by the application, as a
	// [HTTP Verb][httprouter Path string]Route map
	RouteMap map[string]map[string]Route

	// Config is used to create the server struct
	Config struct {
		Routes      RouteMap
		Middlewares []alice.Constructor
	}

	// Server is the struct used to contain all the info as well as helper functions.
	// Port and the return value from GetRouter() are passed into http.ListenAndServe
	Server struct {
		Config
		Router   httprouter.Handle
		Port     string
		router   *httprouter.Router
		handlers alice.Chain
	}
)

// CreateRoute takes an AuthType and a handler, and returns a route struct
func CreateRoute(auth AuthType, handler httprouter.Handle) Route {
	if auth != AUTHTOKEN {
		auth = AUTHNONE
	}

	return Route{
		AuthType: auth,
		Handler:  handler,
	}
}

// GetEmptyRoutes returns an empty RouteMap
func GetEmptyRoutes() RouteMap {
	routes := make(RouteMap)
	verbs := []string{"GET", "PUT", "POST", "OPTIONS", "PATCH", "DELETE"}
	for _, v := range verbs {
		routes[v] = make(map[string]Route)
	}
	return routes
}

func (s *Server) setupRoutes(h alice.Chain) http.Handler {
	m := map[string]interface{}{
		"GET":     s.router.GET,
		"POST":    s.router.POST,
		"PUT":     s.router.PUT,
		"DELETE":  s.router.DELETE,
		"HEAD":    s.router.HEAD,
		"OPTIONS": s.router.OPTIONS,
	}

	for verb, v := range s.Config.Routes {
		for path, route := range v {
			m[verb].(func(string, httprouter.Handle))(path, route.Handler)
		}
	}

	return h.Then(s.router)
}

// GetRouter returns the router for use as the http.Handler argument for
// http.ListenAndServe
func (s *Server) GetRouter() http.Handler {
	return s.setupRoutes(s.handlers)
}

// CreateServer takes a Config struct, and initializes a server
func CreateServer(c Config) Server {
	var port string
	if port = os.Getenv("PORT"); len(port) == 0 {
		port = "8080"
	}

	s := Server{
		Config: c,
		Port:   ":" + port,
		router: httprouter.New(),
	}

	corHandler := cors.New(
		cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowedHeaders:   []string{"Authentication", "Accept", "Content-Type"},
			ExposedHeaders:   []string{"Authentication"},
		},
	)

	h := []alice.Constructor{
		corHandler.Handler,
	}

	if c.Middlewares != nil && len(c.Middlewares) > 0 {
		h = append(h, c.Middlewares...)
	}

	s.handlers = alice.New(h...)

	return s
}
