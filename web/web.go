package web

import (
	"embed"
	"errors"
	"net/http"
	"io"
	"time"
	"log"
	"strings"

        "github.com/tdewolff/minify/v2"
        "github.com/tdewolff/minify/v2/css"
        "github.com/tdewolff/minify/v2/html"
	"github.com/labstack/echo/v4"

	"wzgit/config"
	"wzgit/db"
	"wzgit/csrf"
	"wzgit/httpgit"
)

func unauth(template string, err string) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := struct {
			Error	string
			Captcha	bool
		}{
			Error: err,
			Captcha: config.Cfg.Captcha.Enabled,
		}
		return render(c, template, data)
	}
}

func catchUnauth(f echo.HandlerFunc, template string) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := f(c)
		if err != nil { return unauth(template, err.Error())(c) }
		return nil
	}
}

func redirection(c echo.Context, prefix string, after string) error {
	slash := "/"
	if after == "" { slash = "" }
	return c.Redirect(http.StatusFound, prefix + slash + after)
}

func redirect(c echo.Context, user db.User, after string) error {
	return redirection(c, "/" + user.Name, after)
}

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

func renderCustom(c echo.Context, template string, data any,
			f func(io.Writer) error) error {
	c.Response().WriteHeader(http.StatusOK)
        c.Response().Header().Add("Content-Type", "text/html; charset=utf-8")
        w := minifyHTML(c.Response().Writer)
	user, err := getUser(c)
	navs := strings.Split(c.Request().RequestURI, "/")[1:]
	if navs[len(navs) - 1] == "" { navs = navs[:len(navs) - 1] }
	header := struct {
		Navs		[]string
		Title		string
		CSRF		string
		IsConnected	bool
		Registration	bool
		User		db.User
	}{
		Navs:		navs,
		Title:		config.Cfg.Title,
		CSRF:		csrf.Token(user.Signature),
		IsConnected:	err == nil,
		Registration:	config.Cfg.Users.Registration,
		User:		user,
	}
	err = templates.Lookup("header").Execute(w, header)
	if err != nil { return err }
	err = templates.Lookup(template).Execute(w, data)
	if err != nil { return err }
	if f != nil { if err := f(w); err != nil { return err } }
	err = templates.Lookup("footer").Execute(w, nil)
	if err != nil { return err }
	return w.Close()
}

func render(c echo.Context, template string, data any) error {
	return renderCustom(c, template, data, nil)
}

func getUser(c echo.Context) (db.User, error) {
	cookie, err := c.Cookie("auth_id")
	if err != nil { return db.User{}, err }
	user, exist := db.GetUser(cookie.Value)
	if !exist { return db.User{}, errors.New("user not found") }
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

func Err(next echo.HandlerFunc) echo.HandlerFunc {
	return func (c echo.Context) error {
		err := next(c)
		if err != nil {
			return c.String(http.StatusOK, err.Error())
		}
		return nil
	}
}

func found(dst string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Redirect(http.StatusFound, dst)
	}
}

func dual(unauth echo.HandlerFunc, auth echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, err := getUser(c)
		if err != nil { return unauth(c) }
		return auth(c)
	}
}

func Listen() error {
	e := echo.New()
	e.GET("/robots.txt", static("robots.txt"))
	e.GET("/static/favicon.png", static("favicon.png"))
	e.GET("/static/:path", staticCSS)

	e.GET("/", isConnected(ShowIndex))
	// groups management
	e.GET("/account/groups", acc(ShowGroups))
	e.GET("/account/groups/:group", acc(ShowMembers))
	e.POST("/account/groups/:csrf/addgroup",
		acc(catch(AddGroup, "group_error", "/groups")))
	e.POST("/account/groups/:group/:csrf/desc", acc(SetGroupDesc))
	e.POST("/account/groups/:group/:csrf/add",
		acc(catch(AddToGroup, "group_add_error", "..")))
	e.GET("/account/groups/:group/:csrf/leave", acc(LeaveGroup))
	e.GET("/account/groups/:group/:csrf/delete", acc(DeleteGroup))
	e.GET("/account/groups/:group/:csrf/kick/:user", acc(RmFromGroup))

	// repository settings
	e.GET("/:user/:repo/:csrf/togglepublic", acc(TogglePublic))
	e.POST("/:user/:repo/:csrf/chname", acc(ChangeRepoName))
	e.POST("/:user/:repo/:csrf/chdesc", acc(ChangeRepoDesc))
	e.GET("/:user/:repo/:csrf/delete", acc(DeleteRepo))

	// access management
	e.GET("/:u/:repo/access", acc(ShowAccess))
	e.POST("/:u/:repo/access/:csrf/add",
		acc(catch(AddUserAccess, "access_user_error", "..")))
	e.POST("/:u/:repo/access/:csrf/addg",
		acc(catch(AddGroupAccess, "access_group_error", "..")))
	e.GET("/:u/:repo/access/:user/:csrf/first", acc(UserAccessFirstOption))
	e.GET("/:u/:repo/access/:user/:csrf/second",
		acc(UserAccessSecondOption))
	e.GET("/:u/:repo/access/:group/g/:csrf/first",
		acc(GroupAccessFirstOption))
	e.GET("/:u/:repo/access/:group/g/:csrf/second",
		acc(GroupAccessSecondOption))
	e.GET("/:u/:repo/access/:user/:csrf/kick", acc(RemoveUserAccess))
	e.GET("/:u/:repo/access/:group/g/:csrf/kick", acc(RemoveGroupAccess))

	// user page
	e.POST("/account/:csrf/chdesc", acc(ChangeDesc))
	e.POST("/account/:csrf/addrepo", acc(AddRepo))
	e.GET("/account/:csrf/disconnect", acc(Disconnect))
	e.GET("/account/:csrf/disconnectall", acc(DisconnectAll))
	
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
	// password
	if !config.Cfg.Ldap.Enabled {
		e.POST("/account/passwd/:csrf", acc(catch(ChangePassword,
				"passwd_error", "/account/passwd")))
		e.GET("/account/passwd", acc(ShowPasswd))
	}

	e.GET("/public", PublicList)

	// repository view
	e.GET("/:user/:repo/*", PublicFile)
	e.GET("/:user", PublicAccount)
	e.GET("/:user/:repo", PublicLog)
	e.GET("/:user/:repo/refs", PublicRefs)
	e.GET("/:user/:repo/license", PublicLicense)
	e.GET("/:user/:repo/readme", PublicReadme)
	e.GET("/:user/:repo/files", PublicFiles)
	e.GET("/:user/:repo/files/:blob", PublicFileContent)

	e.GET("/account", acc(ShowAccount))

	e.GET("/login", unauth("login.html", ""))
	e.POST("/login", catchUnauth(Login, "login.html"))
	if config.Cfg.Captcha.Enabled {
		e.GET("/captcha", captchaImage)
	}

	if config.Cfg.Users.Registration {
		e.GET("/register", unauth("register.html", ""))
		e.POST("/register", catchUnauth(Register, "register.html"))
	}

	if config.Cfg.Git.Http.Enabled {
		e.GET("/:user/:repo/info/refs",
			echo.WrapHandler(httpgit.Handle(config.Cfg.Git.Path)))
		e.POST("/:user/:repo/git-upload-pack",
			echo.WrapHandler(httpgit.Handle(config.Cfg.Git.Path)))
		e.POST("/:user/:repo/git-receive-pack",
			echo.WrapHandler(httpgit.Handle(config.Cfg.Git.Path)))
	}

	e.Use(Logger)
	e.Use(Err)

	return e.Start(config.Cfg.Web.Host)
}
