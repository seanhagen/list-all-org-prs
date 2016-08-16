package server

import (
	"fmt"
	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"github.com/markbates/goth/gothic"
	"io"
	"net/http"
)

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
		_, err = io.WriteString(w, fmt.Sprintf("user: %#v", user))
		if err != nil {
			panic(err)
		}
	}
}

func buildIndexRoute() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		_, err := io.WriteString(w, "hey, okay!")
		if err != nil {
			panic(err)
		}
	}
}
