package server

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	// gctx "github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	// "github.com/markbates/goth/gothic"
	"golang.org/x/net/context"
	// gh "github.com/markbates/goth/providers/github"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"io"
	"net/http"
	// "net/url"
	"github.com/satori/go.uuid"
	// "golang.org/x/oauth2"
	"bytes"
	"google.golang.org/appengine/memcache"
	"strings"
)

var paramContext = "params"
var cookieName = "magic-thing"

var githubURL = "https://api.github.com"

func getTokenFromMemcached(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return "", err
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

func buildProviderRoute(s *Server) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := appengine.NewContext(r)
		checkForToken(ctx, w, r)

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
		checkForToken(ctx, w, r)

		client := urlfetch.Client(ctx)

		http.DefaultClient = client
		http.DefaultTransport = client.Transport

		token, err := s.oauth.Exchange(ctx, r.URL.Query().Get("code"))

		if err != nil {
			_, _ = io.WriteString(w, fmt.Sprintf("error fetching token: %#v", err))
			return
		}

		randomID := uuid.NewV4()
		item := &memcache.Item{
			Key:   randomID.String(),
			Value: []byte(token.AccessToken),
		}
		err = memcache.Add(ctx, item)
		if err != nil {
			_, _ = io.WriteString(w, fmt.Sprintf("Unable to write key to memcached: %v", err))
			return
		}

		cookie := &http.Cookie{
			Name:     cookieName,
			Value:    randomID.String(),
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func buildIndexRoute() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := appengine.NewContext(r)
		client := urlfetch.Client(ctx)

		token, err := getTokenFromMemcached(ctx, w, r)

		if err != nil {
			if strings.Contains(r.Referer(), "github-pr-list") {
				o := fmt.Sprintf("got here from %#v<br/><br/>token: %#v<br/><br/>error: %#v", r.Referer(), token, err)
				_, _ = io.WriteString(w, o)
			} else {
				http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
			}
		} else {
			_, _ = io.WriteString(w, fmt.Sprintf("got token: %#v", token))
		}

		url := githubURL + "/issues?filter=all"
		req, err := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "token "+token)
		req.Header.Set("User-Agent", "golang-http-client")

		var results []*github.Issue

		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			_, _ = io.WriteString(w, fmt.Sprintf("error fetching issues: %#v<br/><br/>resp: %#v", err, resp))
			_, _ = io.WriteString(w, fmt.Sprintf("<br/><br/><br/>headers: %#v", resp.Header))
			return
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		json.Unmarshal(buf.Bytes(), results)

		_, _ = io.WriteString(w, buf.String())
		_, _ = io.WriteString(w, fmt.Sprintf("<br/><br/>response: <br/><br/>%#v", results))
	}
}
