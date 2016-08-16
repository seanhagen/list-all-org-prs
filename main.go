package main

import (
	"fmt"
	"github.com/seanhagen/list-all-org-prs/server"
	"log"
	"net/http"
)

func main() {
	s := server.CreateServer()
	fmt.Printf("Starting server on port %#v\n", s.Port)
	log.Fatal(http.ListenAndServe(":"+s.Port, s.Router))
}
