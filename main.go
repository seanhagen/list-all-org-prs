package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/seanhagen/list-all-org-prs/server"
	"io"
	"log"
	"net/http"
)

func main() {
	c := server.Config{
		Routes: buildRoutes(),
	}

	s := server.CreateServer(c)

	log.Fatal(http.ListenAndServe(s.Port, s.GetRouter()))
}

func buildRoutes() server.RouteMap {
	r := server.GetEmptyRoutes()
	r["GET"]["/"] = server.CreateRoute(server.AUTHNONE, buildIndexRoute())
	return r
}

func buildIndexRoute() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "hey, okay!")
	}
}
