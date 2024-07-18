package web

import (
	_ "embed"
	"errors"
	"net/http"
	"github.com/labstack/echo/v4"

	"gemigit/config"
	"gemigit/gmi"
	"gemigit/db"
	"gemigit/csrf"
)

//go:embed templates/robots.txt
var robots string

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
	if err := csrf.Handle(user, c.Param("csrf")); err != nil {
		return db.User{}, err
	}
	return user, nil
}

func isConnected(f func(echo.Context, bool) error) func(c echo.Context) error {
	return func(c echo.Context) error {
		_, err := getUser(c)
		return f(c, err == nil)
	}
}

func auth(f func(echo.Context, db.User) error) func(c echo.Context) error {
	return func(c echo.Context) error {
		user, err := getUser(c)
		if err != nil {
			//return c.String(http.StatusBadRequest, err.Error())
			//c.Logger().Info(err)
			//c.Logger().Info("test")
			return c.Redirect(http.StatusFound, "/asd")
		}
		if err := f(c, user); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return nil
	}

}

func Listen() {
	e := echo.New()
	e.GET("/robots.txt", func(c echo.Context) error {
		return c.String(http.StatusOK, robots)
	})
/*
	if config.Cfg.Gemini.StaticDirectory != "" {
		g.Static("/static", config.Cfg.Gemini.StaticDirectory)
	}

	//passAuth := gig.PassAuth(csrf.Handle)

	secure := g.Group("/account/", passAuth)
	g := e.Group("/account/",
		func(next echo.HandlerFunc) echo.HandlerFunc {
			
		})

	echo.Mid
	g.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		if username == "joe" && password == "secret" {
			return true, nil
		}
		return false, nil
	}))
	*/

	e.GET("/account", auth(gmi.ShowAccount))
	// groups management
	e.GET("/account/groups", auth(gmi.ShowGroups))
	e.GET("/account/groups/:group", auth(gmi.ShowMembers))
	e.GET("groups/:group/:csrf/desc", auth(gmi.SetGroupDesc))
	e.GET("groups/:group/:csrf/desc", auth(gmi.SetGroupDesc))
	e.GET("groups/:group/:csrf/add", auth(gmi.AddToGroup))
	e.GET("groups/:group/:csrf/leave", auth(gmi.LeaveGroup))
	e.GET("groups/:group/:csrf/delete", auth(gmi.DeleteGroup))
	e.GET("groups/:group/:csrf/kick/:user", auth(gmi.RmFromGroup))

	// repository settings
	e.GET("repo/:repo/*", auth(gmi.RepoFile))
	e.GET("repo/:repo/:csrf/togglepublic", auth(gmi.TogglePublic))
	e.GET("repo/:repo/:csrf/chname", auth(gmi.ChangeRepoName))
	e.GET("repo/:repo/:csrf/chdesc", auth(gmi.ChangeRepoDesc))
	e.GET("repo/:repo/:csrf/delrepo", auth(gmi.DeleteRepo))

	// access management
	e.GET("repo/:repo/access", auth(gmi.ShowAccess))
	e.GET("repo/:repo/access/:csrf/add", auth(gmi.AddUserAccess))
	e.GET("repo/:repo/access/:csrf/addg", auth(gmi.AddGroupAccess))
	e.GET("repo/:repo/access/:user/:csrf/first",
		auth(gmi.UserAccessFirstOption))
	e.GET("repo/:repo/access/:user/:csrf/second",
		auth(gmi.UserAccessSecondOption))
	e.GET("repo/:repo/access/:group/g/:csrf/first",
		auth(gmi.GroupAccessFirstOption))
	e.GET("repo/:repo/access/:group/g/:csrf/second",
		auth(gmi.GroupAccessSecondOption))
	e.GET("repo/:repo/access/:user/:csrf/kick",
		auth(gmi.RemoveUserAccess))
	e.GET("repo/:repo/access/:group/g/:csrf/kick",
		auth(gmi.RemoveGroupAccess))

	// repository view
	e.GET("repo/:repo", auth(gmi.RepoLog))
	/*e.GET("repo/:repo/", func(c gig.Context) error {
		return c.NoContent(gig.StatusRedirectTemporary,
			"/account/repo/" + c.Param("repo"))
	})*/
	e.GET("repo/:repo/license", auth(gmi.RepoLicense))
	e.GET("repo/:repo/readme", auth(gmi.RepoReadme))
	e.GET("repo/:repo/refs", auth(gmi.RepoRefs))
	e.GET("repo/:repo/files", auth(gmi.RepoFiles))
	e.GET("repo/:repo/files/:blob", auth(gmi.RepoFileContent))

	// user page
	e.GET("/account/:csrf/chdesc", auth(gmi.ChangeDesc))
	e.GET("/account/:csrf/addrepo", auth(gmi.AddRepo))
	e.GET("/account/:csrf/addgroup", auth(gmi.AddGroup))
	e.GET("/account/:csrf/disconnect", auth(gmi.Disconnect))
	e.GET("/account/:csrf/disconnectall", auth(gmi.DisconnectAll))
	if !config.Cfg.Ldap.Enabled {
		e.GET("/account/:csrf/chpasswd", auth(gmi.ChangePassword))
	}
	// otp
	e.GET("/account/otp", auth(gmi.ShowOTP))
	e.GET("/account/otp/:csrf/qr", auth(gmi.CreateTOTP))
	e.GET("/account/otp/:csrf/confirm", auth(gmi.ConfirmTOTP))
	e.GET("/account/otp/:csrf/rm", auth(gmi.RemoveTOTP))
	// token
	e.GET("/account/token", auth(gmi.ShowTokens))
	e.GET("/account/token/:csrf/new", auth(gmi.CreateWriteToken))
	e.GET("/account/token/:csrf/new_ro", auth(gmi.CreateReadToken))
	e.GET("/account/token/:csrf/secure", auth(gmi.ToggleTokenAuth))
	e.GET("/account/token/:csrf/renew/:token", auth(gmi.RenewToken))
	e.GET("/account/token/:csrf/delete/:token", auth(gmi.DeleteToken))

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

	e.GET("/repo", gmi.PublicList)
	e.GET("/repo/:user/:repo/*", gmi.PublicFile)
	e.GET("/repo/:user", gmi.PublicAccount)
	e.GET("/repo/:user/:repo", gmi.PublicLog)
	e.GET("/repo/:user/:repo/refs", gmi.PublicRefs)
	e.GET("/repo/:user/:repo/license", gmi.PublicLicense)
	e.GET("/repo/:user/:repo/readme", gmi.PublicReadme)
	e.GET("/repo/:user/:repo/files", gmi.PublicFiles)
	e.GET("/repo/:user/:repo/files/:blob", gmi.PublicFileContent)

	e.POST("/login", errorPage(gmi.Login))
	//g.PassAuthLoginHandle("/login", gmi.Login)

	if config.Cfg.Users.Registration {
		e.POST("/register", errorPage(gmi.Register))
	}
	e.GET("/otp", auth(gmi.LoginOTP))

	e.GET("/", isConnected(gmi.ShowIndex))

	e.Logger.Fatal(e.Start(config.Cfg.Web.Host))
}
