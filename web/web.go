package web

import (
	"embed"
	"errors"
	"net/http"
	"io"
	"time"
	"log"

        "github.com/tdewolff/minify/v2"
        "github.com/tdewolff/minify/v2/css"
        "github.com/tdewolff/minify/v2/html"
	"github.com/labstack/echo/v4"

	"gemigit/config"
	"gemigit/db"
	"gemigit/csrf"
	"gemigit/httpgit"
)

//go:embed static/*
var staticFS embed.FS

//go:embed css/*
var cssFS embed.FS

func static(path string) echo.HandlerFunc {
	return func(c echo.Context) error {
		f, err := staticFS.Open("static/" + path)
		if err != nil {
			return c.String(http.StatusNotFound, err.Error())
		}
		_, err = io.Copy(c.Response().Writer, f)
		return err
	}
}

var cachedCSS = map[string][]byte{}
func staticCSS(c echo.Context) error {
	name := c.Param("path")
	if data, ok := cachedCSS[name]; ok {
		return c.Blob(http.StatusOK, "text/css", data)
	}
	data, err := cssFS.ReadFile("css/" + c.Param("path"))
	if err != nil {
		return c.String(http.StatusNotFound, err.Error())
	}
	data, err = minifyCSS(data)
	if err != nil { return err }
	cachedCSS[name] = data
	c.Response().Header().Add("Content-Type", "text/css")
	return c.Blob(http.StatusOK, "text/css", data)
}

func minifyCSS(in []byte) ([]byte, error) {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	res, err := m.Bytes("text/css", in)
	if err != nil { return nil, err }
	return res, nil
}

func minifyHTML(w io.Writer) io.WriteCloser {
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	return m.Writer("text/html", w)
}

func render(c echo.Context, template string, data any) error {
        c.Response().WriteHeader(http.StatusOK)
        c.Response().Header().Add("Content-Type", "text/html; charset=utf-8")
        w := minifyHTML(c.Response().Writer)
	user, err := getUser(c)
	header := struct {
		Title		string
		IsConnected	bool
		User		db.User
	}{
		Title:		config.Cfg.Title,
		IsConnected:	err == nil,
		User:		user,
	}
        err = templates.Lookup("header").Execute(w, header)
        if err != nil { return err }
        err = templates.Lookup(template).Execute(w, data)
        if err != nil { return err }
        err = templates.Lookup("footer").Execute(w, nil)
        if err != nil { return err }
        return w.Close()
}

func errorPage(f func(echo.Context) error) func(c echo.Context) error {
	return func(c echo.Context) error {
		if err := f(c); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return nil
	}
}

func getUser(c echo.Context) (db.User, error) {
	cookie, err := c.Cookie("auth_id")
	if err != nil {
		return db.User{}, err
	}
	user, exist := db.GetUser(cookie.Value)
	if !exist {
		return db.User{}, errors.New("user not found")
	}
	renew := c.Request().Method == "POST"
	if err := csrf.Handle(user, c.Param("csrf"), renew); err != nil {
		return db.User{}, err
	}
	return user, nil
}

func isConnected(f func(echo.Context, bool) error) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, err := getUser(c)
		return f(c, err == nil)
	}
}

func acc(f func(echo.Context, db.User) error) func(c echo.Context) error {
	return func(c echo.Context) error {
		user, err := getUser(c)
		if err != nil {
			return c.Redirect(http.StatusFound, "/")
		}
		if err := f(c, user); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return nil
	}
}

func Logger(next echo.HandlerFunc) echo.HandlerFunc {
	return func (c echo.Context) error {
		t1 := time.Now()
		err := next(c)
		t2 := time.Now()
		r := c.Request()
		realIP := r.Header.Get("X-Real-IP")
		if realIP == "" {
			realIP = r.RemoteAddr
		}
		log.Println("["+realIP+"]["+r.Method+"]",
			    r.URL.String(), t2.Sub(t1))
		return err
	}
}

func Listen() {
	e := echo.New()
	e.GET("/robots.txt", static("robots.txt"))
	e.GET("/css/:path", staticCSS)

	e.GET("/account", acc(ShowAccount))
	// groups management
	e.GET("/account/groups", acc(ShowGroups))
	e.GET("/account/groups/:group", acc(ShowMembers))
	e.POST("/account/groups/:csrf/addgroup", acc(catch(AddGroup,
		"group_error", "/groups")))
	e.GET("/account/groups/:group/:csrf/desc", acc(SetGroupDesc))
	e.GET("/account/groups/:group/:csrf/desc", acc(SetGroupDesc))
	e.GET("/account/groups/:group/:csrf/add", acc(AddToGroup))
	e.GET("/account/groups/:group/:csrf/leave", acc(LeaveGroup))
	e.GET("/account/groups/:group/:csrf/delete", acc(DeleteGroup))
	e.GET("/account/groups/:group/:csrf/kick/:user", acc(RmFromGroup))

	// repository settings
	e.GET("/account/repo/:repo/*", acc(RepoFile))
	e.GET("/account/repo/:repo/:csrf/togglepublic", acc(TogglePublic))
	e.GET("/account/repo/:repo/:csrf/chname", acc(ChangeRepoName))
	e.GET("/account/repo/:repo/:csrf/chdesc", acc(ChangeRepoDesc))
	e.GET("/account/repo/:repo/:csrf/delrepo", acc(DeleteRepo))

	// access management
	e.GET("/account/repo/:repo/access", acc(ShowAccess))
	e.GET("/account/repo/:repo/access/:csrf/add", acc(AddUserAccess))
	e.GET("/account/repo/:repo/access/:csrf/addg", acc(AddGroupAccess))
	e.GET("/account/repo/:repo/access/:user/:csrf/first",
		acc(UserAccessFirstOption))
	e.GET("/account/repo/:repo/access/:user/:csrf/second",
		acc(UserAccessSecondOption))
	e.GET("/account/repo/:repo/access/:group/g/:csrf/first",
		acc(GroupAccessFirstOption))
	e.GET("/account/repo/:repo/access/:group/g/:csrf/second",
		acc(GroupAccessSecondOption))
	e.GET("/account/repo/:repo/access/:user/:csrf/kick",
		acc(RemoveUserAccess))
	e.GET("/account/repo/:repo/access/:group/g/:csrf/kick",
		acc(RemoveGroupAccess))

	// repository view
	e.GET("/account/repo/:repo", acc(RepoLog))
	/*e.GET("repo/:repo/", func(c gig.Context) error {
		return c.NoContent(gig.StatusRedirectTemporary,
			"/account/repo/" + c.Param("repo"))
	})*/
	e.GET("/account/repo/:repo/license", acc(RepoLicense))
	e.GET("/account/repo/:repo/readme", acc(RepoReadme))
	e.GET("/account/repo/:repo/refs", acc(RepoRefs))
	e.GET("/account/repo/:repo/files", acc(RepoFiles))
	e.GET("/account/repo/:repo/files/:blob", acc(RepoFileContent))

	// user page
	e.POST("/account/:csrf/chdesc", acc(ChangeDesc))
	e.POST("/account/:csrf/addrepo", acc(AddRepo))
	e.GET("/account/:csrf/disconnect", acc(Disconnect))
	e.GET("/account/:csrf/disconnectall", acc(DisconnectAll))
	if !config.Cfg.Ldap.Enabled {
		e.POST("/account/:csrf/chpasswd",
			acc(catch(ChangePassword, "chpasswd_error", "")))
	}
	// otp
	e.GET("/account/otp", acc(ShowOTP))
	e.GET("/account/otp/:csrf/qr", acc(CreateTOTP))
	e.POST("/account/otp/:csrf/confirm", acc(ConfirmTOTP))
	e.POST("/account/otp/:csrf/rm", acc(RemoveTOTP))
	// token
	e.GET("/account/token", acc(ShowTokens))
	e.GET("/account/token/:csrf/new", acc(CreateWriteToken))
	e.GET("/account/token/:csrf/new_ro", acc(CreateReadToken))
	e.GET("/account/token/:csrf/secure", acc(ToggleTokenAuth))
	e.GET("/account/token/:csrf/renew/:token", acc(RenewToken))
	e.GET("/account/token/:csrf/delete/:token", acc(DeleteToken))

	/*if config.Cfg.Git.Public {
		public = g.Group("/repo")
	} else {
		public = g.Group("/repo", gig.PassAuth(
			func(sig string, c gig.Context) (string, error) {
				_, exist := db.GetUser(sig)
				if !exist { return "/", nil }
				return "", nil
			}))
	}*/

	e.GET("/repo", PublicList)
	e.GET("/repo/:user/:repo/*", PublicFile)
	e.GET("/repo/:user", PublicAccount)
	e.GET("/repo/:user/:repo", PublicLog)
	e.GET("/repo/:user/:repo/refs", PublicRefs)
	e.GET("/repo/:user/:repo/license", PublicLicense)
	e.GET("/repo/:user/:repo/readme", PublicReadme)
	e.GET("/repo/:user/:repo/files", PublicFiles)
	e.GET("/repo/:user/:repo/files/:blob", PublicFileContent)

	e.POST("/login", errorPage(Login))

	if config.Cfg.Users.Registration {
		e.POST("/register", errorPage(Register))
	}
	e.GET("/", isConnected(ShowIndex))

	if config.Cfg.Git.Http.Enabled {
		e.GET("/git/:user/:repo",
			echo.WrapHandler(httpgit.Listen(config.Cfg.Git.Path)))
		e.POST("/git/:user/:repo",
			echo.WrapHandler(httpgit.Listen(config.Cfg.Git.Path)))
	}

	e.Use(Logger)

	e.Logger.Fatal(e.Start(config.Cfg.Web.Host))
}
