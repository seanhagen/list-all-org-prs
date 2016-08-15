package main

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/seanhagen/list-all-org-prs/server"
	"io"
	"log"
	"net/http"
	"os"
)

func init() {
	log.SetOutput(os.Stdout)
	gothic.Store = sessions.NewFilesystemStore(os.TempDir(), []byte("list-all-org-prs"))
}

func main() {
	goth.UseProviders(
		github.New(
			os.Getenv("GITHUB_KEY"),
			os.Getenv("GITHUB_SECRET"),
			"http://localhost:8080/auth/github/callback",
		),
	)

	c := server.Config{
		Routes: buildRoutes(),
	}
	s := server.CreateServer(c)
	s.Start()
}

func buildRoutes() server.RouteMap {
	r := server.GetEmptyRoutes()
	r["GET"]["/"] = server.CreateRoute(server.AUTHNONE, buildIndexRoute())
	r["GET"]["/auth/:provider/callback"] = server.CreateRoute(server.AUTHNONE, buildCallbackRoute())
	r["GET"]["/auth/:provider"] = server.CreateRoute(server.AUTHNONE, buildProviderRoute())
	return r
}

func setParamsInContext(r *http.Request, p httprouter.Params) {
	context.Set(r, "params", p)
}

func getParamsFromContext(r *http.Request) httprouter.Params {
	return context.Get(r, "params").(httprouter.Params)
}

func getProviderName(r *http.Request) (string, error) {
	params := getParamsFromContext(r)
	p := params.ByName("provider")

	return p, nil
}

func buildProviderRoute() httprouter.Handle {
	gothic.GetProviderName = getProviderName
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		setParamsInContext(r, params)
		gothic.BeginAuthHandler(w, r)
	}
}

func buildCallbackRoute() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		user, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		io.WriteString(w, fmt.Sprintf("user: %#v", user))
	}
}

func buildIndexRoute() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "hey, okay!")
	}
}
