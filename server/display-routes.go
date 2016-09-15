package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
	"github.com/russross/blackfriday"
	"github.com/shurcooL/github_flavored_markdown"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"html/template"
	"io"
	"net/http"
	"strings"
)

var (
	githubURL = "https://api.github.com"

	baseTemplate = `<!DOCTYPE html>
<html lang="en">
  <head>
    <title>{{ template "title" . }}</title>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
    <link rel="stylesheet" href="/static/css/bootstrap.min.css" >
    <link rel="stylesheet" href="/static/css/font-awesome.css" >
    <link rel="stylesheet" href="/static/css/bootstrap-social.css" >
    <link rel="stylesheet" href="/static/css/main.css" >
  </head>
  <body>
    <div class="container">
      <div class="header clearfix">
        <nav>
          <ul class="nav nav-pills pull-right">
            <li role="presentation"><a href="https://github.com/seanhagen">Contact</a></li>
          </ul>
        </nav>
        <h3 class="text-muted">GitHub PR Lister</h3>
      </div>

     	{{ template "content" . }}

      <footer class="footer">
        <p>This is just a silly thing, for the source check out the <a href="https://github.com/seanhagen/list-all-org-prs">repo</a></p>
      </footer>
    </div>
  </body>
  <script src="https://ajax.googleapis.com/ajax/libs/jquery/1.12.4/jquery.min.js"></script>
  <script src="/static/js/bootstrap.min.js"></script>
</html>`

	loginTemplate = `{{ define "title" }}GitHub PR Lister - Login{{ end }}
{{ define "content"}}
      <div class="jumbotron">
        <p class="lead">
          To get started, log in with GitHub!
          <a id="login-with-github"
             class="btn btn-block btn-social btn-lg btn-github"
             href="/auth">
            <i class="fa fa-github"></i>Sign in with GitHub
          </a>
        </p>
      </div>
{{ end }}`

	listPageTemplate = `{{ define "title" }}GitHub PR Lister - Login{{ end }}
{{ define "content"}}
<div class="row">
  <div class="col-lg-8">
    {{range .Issues}}

    <div class="panel-group" id="accordion" role="tablist" aria-multiselectable="true">
      <div class="panel panel-default">
        <div class="panel-heading" role="tab" id="headingOne">
          <h4 class="panel-title">
            <a role="button" data-toggle="collapse" data-parent="#accordion" href="#collapse{{.Number}}" aria-expanded="true" aria-controls="collapse{{.Number}}">
              {{.Repository.Owner.Login}}/{{.Repository.Name}} - #{{.Number}}: {{.Title}}
            </a>

            <span class="text-right">
              <a target="_blank" href="{{.HTMLURL}}">Go to PR</a>
            </span>
          </h4>
        </div>
        <div id="collapse{{.Number}}" class="panel-collapse collapse" role="tabpanel" aria-labelledby="headingOne" aria-expanded="false">
          <div class="panel-body">
            {{md .Body }}
          </div>
        </div>
      </div>
    </div>
    {{end}}
  </div>
</div>
{{ end }}`
)

func buildLoginDisplayRoute() httprouter.Handle {
	base, err := template.New("base").Parse(baseTemplate)
	if err != nil {
		panic(err)
	}

	login, err := template.Must(base.Clone()).Parse(loginTemplate)
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		err := login.Execute(w, nil)
		if err != nil {
			_, _ = io.WriteString(w, fmt.Sprintf("Error executing template: %v", err))
		}
	}
}

func getNextPageLink(h http.Header) string {
	if links, ok := h["Link"]; ok && len(links) > 0 {
		for _, link := range strings.Split(links[0], ",") {
			segments := strings.Split(strings.TrimSpace(link), ";")

			if len(segments) < 2 {
				continue
			}

			// ensure href is properly formatted
			if !strings.HasPrefix(segments[0], "<") || !strings.HasSuffix(segments[0], ">") {
				continue
			}

			if strings.TrimSpace(segments[1]) == `rel="next"` {
				return strings.TrimSuffix(strings.TrimPrefix(segments[0], "<"), ">")
			}
		}
	}

	return ""
}

func fetchIssuesFromURL(ctx context.Context, token, url string) ([]*github.Issue, http.Header, error) {
	client := urlfetch.Client(ctx)
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("User-Agent", "golang-http-client")
	resp, err := client.Do(req)
	defer func() { _ = resp.Body.Close() }()

	var results []*github.Issue

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	err = json.Unmarshal(buf.Bytes(), &results)
	// err = json.NewDecoder(resp.Body).Decode(&results)

	if err != nil {
		err = fmt.Errorf("Unable to decode body: %#v ( URL: %v - Token: %v )", buf.String(), url, token)
	}

	return results, resp.Header, err
}

func fetchIssues(ctx context.Context, token string) ([]*github.Issue, error) {
	url := githubURL + "/issues?filter=all&per_page=100"
	log.Infof(ctx, "Fetching first page of issues, using url: %v", url)

	var results []*github.Issue

	for {
		res, header, err := fetchIssuesFromURL(ctx, token, url)
		if err != nil {
			return results, err
		}
		log.Infof(ctx, "Got some results")

		for _, r := range res {
			if r.PullRequestLinks != nil {
				results = append(results, r)
			}
		}

		if nextLink := getNextPageLink(header); len(nextLink) > 0 {
			log.Infof(ctx, "Got a 'next page' link: %v", nextLink)
			url = nextLink
			continue
		}
		break
	}

	return results, nil
}

func markdownIt(in string) template.HTML {
	input := []byte(in)
	unsafe := blackfriday.MarkdownCommon(input)
	return template.HTML(github_flavored_markdown.Markdown(unsafe))
}

func buildIndexRoute() httprouter.Handle {
	funcs := template.FuncMap{"md": markdownIt}
	base, _ := template.New("base").Funcs(funcs).Parse(baseTemplate)
	index, _ := template.Must(base.Clone()).Parse(listPageTemplate)

	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := appengine.NewContext(r)

		token, err := getTokenFromMemcached(ctx, w, r)

		if err != nil || token == "" {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		results, err := fetchIssues(ctx, token)
		if err != nil {
			_, _ = io.WriteString(w, fmt.Sprintf("Unable to fetch issues: %#v", err))
			return
		}

		data := struct {
			Issues []*github.Issue
		}{
			Issues: results,
		}

		err = index.Execute(w, data)
		if err != nil {
			_, _ = io.WriteString(w, fmt.Sprintf("Error executing template: %v", err))
		}
	}
}
