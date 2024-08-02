package httpgit

import (
	"wzgit/access"
	"wzgit/config"
	"wzgit/db"
	"log"
	"net/http"
	"strings"
	//"net/url"

	githttpxfer "github.com/nulab/go-git-http-xfer/githttpxfer"
)

func Handle(path string) http.Handler {
	ghx, err := githttpxfer.New(path, "git")
	if err != nil {
		log.Fatalln("GitHTTPXfer instance could not be created. ",
			    err.Error())
	}

	chain := newChain()
	chain.use(basicAuth)
	handler := chain.build(ghx)

	return handler
}

type middleware func(http.Handler) http.Handler

func newChain() *chain {
	return &chain{[]middleware{}}
}

type chain struct {
	middlewares []middleware
}

func (c *chain) use(m middleware) {
	c.middlewares = append(c.middlewares, m)
}

func (c *chain) build(h http.Handler) http.Handler {
	for i := range c.middlewares {
		h = c.middlewares[len(c.middlewares)-1-i](h)
	}
	return h
}

func basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		readOnly := false
		public := false
		params := strings.Split(r.URL.Path[1:], "/")
		if len(params) < 2 {
			renderNotFound(w)
			return
		}
		owner := params[0]
		repo := params[1]
		log.Println(owner, repo, r.URL.RequestURI())
		if strings.Contains(r.URL.Path, "git-upload-pack") ||
		   strings.Contains(r.URL.RawQuery, "git-upload-pack") {
			readOnly = true
			public, err = db.IsRepoPublic(repo, owner)
			if err != nil {
				log.Println(err)
				renderNotFound(w)
				return
			}
			if config.Cfg.Git.Public && public {
				next.ServeHTTP(w, r)
				return
			}
		}

		username, password, ok := r.BasicAuth()
		if !ok {
			renderUnauthorized(w)
			return
		}
		/* check if it is allowed to use password instead of token */
		pass, err := db.CanUsePassword(repo, owner, username)
		if err != nil {
			log.Println(err.Error())
			renderUnauthorized(w)
			return
		}
		err = access.Login(username, password, true, pass, !readOnly)
		if err != nil {
			log.Println(err.Error())
			renderUnauthorized(w)
			return
		}
		if readOnly && public {
			next.ServeHTTP(w, r)
			return
		}
		if readOnly {
			err = access.HasReadAccess(repo, owner, username)
		} else {
			err = access.HasWriteAccess(repo, owner, username)
		}
		if err != nil {
			log.Println(err.Error())
			renderUnauthorized(w)
			return
		}
		log.Println("next HTTP")
		next.ServeHTTP(w, r)
	})
}

func renderNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(http.StatusText(http.StatusNotFound)))
	w.Header().Set("Content-Type", "text/plain")
}

func renderUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Basic realm=\"" +
		       "Please enter your username and password.\"")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
	w.Header().Set("Content-Type", "text/plain")
}
