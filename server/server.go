package server

import (
	// "github.com/gorilla/sessions"
	gh "github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/markbates/goth/providers/github"
	"github.com/rs/cors"
	"golang.org/x/oauth2"
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

	// Server is the struct used to contain all the info as well as helper functions.
	// Port and the return value from GetRouter() are passed into http.ListenAndServe
	Server struct {
		Routes      RouteMap
		Router      http.Handler
		Port        string
		router      *httprouter.Router
		handlers    alice.Chain
		oauth       *oauth2.Config
		middlewares []alice.Constructor
	}
)

func createRoute(auth AuthType, handler httprouter.Handle) Route {
	if auth != AUTHTOKEN {
		auth = AUTHNONE
	}

	return Route{
		AuthType: auth,
		Handler:  handler,
	}
}

func getEmptyRoutes() RouteMap {
	routes := make(RouteMap)
	verbs := []string{"GET", "PUT", "POST", "OPTIONS", "PATCH", "DELETE"}
	for _, v := range verbs {
		routes[v] = make(map[string]Route)
	}
	return routes
}

func (s *Server) setupOauth() {
	s.oauth = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_KEY"),
		ClientSecret: os.Getenv("GITHUB_SECRET"),
		Scopes: []string{
			string(gh.ScopeReadOrg),
			string(gh.ScopePublicRepo),
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  github.AuthURL,
			TokenURL: github.TokenURL,
		},
	}
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

	for verb, v := range s.Routes {
		for path, route := range v {
			m[verb].(func(string, httprouter.Handle))(path, route.Handler)
		}
	}

	return h.Then(s.router)
}

func (s *Server) buildRoutes() {
	r := getEmptyRoutes()
	r["GET"]["/"] = createRoute(AUTHNONE, buildIndexRoute())
	r["GET"]["/auth/callback"] = createRoute(AUTHNONE, buildCallbackRoute(s))
	r["GET"]["/auth"] = createRoute(AUTHNONE, buildAuthRoute(s))
	r["GET"]["/login"] = createRoute(AUTHNONE, buildLoginDisplayRoute())

	s.Routes = r
}

func (s *Server) setupMiddlewares() {
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
		tokenAuth(s),
		corHandler.Handler,
	}

	if s.middlewares != nil && len(s.middlewares) > 0 {
		h = append(h, s.middlewares...)
	}

	s.middlewares = h
}

// CreateServer takes a Config struct, and initializes a server
func CreateServer() Server {
	var port string
	if port = os.Getenv("PORT"); len(port) == 0 {
		port = "8080"
	}

	s := Server{
		Port:   port,
		router: httprouter.New(),
	}
	s.buildRoutes()
	s.setupOauth()
	s.setupMiddlewares()

	handler := alice.New(s.middlewares...)
	s.Router = s.setupRoutes(handler)
	return s
}
