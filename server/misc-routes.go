package server

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/urlfetch"
	"io"
	"net/http"
	"time"
)

var cookieName = "magic-thing"

func getTokenFromMemcached(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return c.String(), err
	}

	item, err := memcache.Get(ctx, c.Value)
	if err != nil {
		return "", err
	}

	return string(item.Value), nil
}

func checkForToken(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, err := getTokenFromMemcached(ctx, w, r)
	if err == nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func buildAuthRoute(s *Server) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := appengine.NewContext(r)
		client := urlfetch.Client(ctx)
		http.DefaultClient = client
		http.DefaultTransport = client.Transport
		url := s.oauth.AuthCodeURL("state")
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func buildCallbackRoute(s *Server) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := appengine.NewContext(r)
		client := urlfetch.Client(ctx)

		http.DefaultClient = client
		http.DefaultTransport = client.Transport

		token, err := s.oauth.Exchange(ctx, r.URL.Query().Get("code"))

		if err != nil {
			_, _ = io.WriteString(w, fmt.Sprintf("error fetching token: %#v", err))
			return
		}

		randomID := uuid.NewV4().String()
		item := &memcache.Item{
			Key:   randomID,
			Value: []byte(token.AccessToken),
		}
		err = memcache.Add(ctx, item)
		if err != nil {
			_, _ = io.WriteString(w, fmt.Sprintf("Unable to write key to memcached: %v", err))
			return
		}
		cookie := &http.Cookie{
			Name:     cookieName,
			Value:    randomID,
			Path:     "/",
			Domain:   "github-pr-list.appspot.com",
			Expires:  time.Now().Add(time.Hour * 24),
			MaxAge:   int(time.Now().Add(time.Hour * 24).Unix()),
			Secure:   true,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
}
