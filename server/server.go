package server

import (
	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
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

	// Server is the struct used to contain all the info as well as helper functions.
	// Port and the return value from GetRouter() are passed into http.ListenAndServe
	Server struct {
		Routes      RouteMap
		Router      http.Handler
		Port        string
		router      *httprouter.Router
		handlers    alice.Chain
		middlewares []alice.Constructor
	}
)

func init() {
	gothic.Store = sessions.NewFilesystemStore(os.TempDir(), []byte("list-all-org-prs"))
}

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

func (s *Server) setupGothic() {
	goth.UseProviders(
		github.New(
			os.Getenv("GITHUB_KEY"),
			os.Getenv("GITHUB_SECRET"),
			"http://localhost:8080/auth/github/callback",
		),
	)
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
	r["GET"]["/auth/:provider/callback"] = createRoute(AUTHNONE, buildCallbackRoute())
	r["GET"]["/auth/:provider"] = createRoute(AUTHNONE, buildProviderRoute())

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
		logMiddleware,
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
	s.setupGothic()
	s.setupMiddlewares()

	handler := alice.New(s.middlewares...)
	s.Router = s.setupRoutes(handler)
	return s
}
