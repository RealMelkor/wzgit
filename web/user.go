package web

import (
	"time"
	"net/http"
        "gemigit/db"
        "gemigit/repo"
        "gemigit/config"

	"github.com/labstack/echo/v4"
)

func ChangeDesc(c echo.Context, user db.User) error {
	newdesc := c.QueryString()
	if err := user.ChangeDescription(newdesc);
	   err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return redirect(c, "")
}

func AddRepo(c echo.Context, user db.User) error {
	name := c.QueryString()
	if err := user.CreateRepo(name);
	   err != nil {
		return c.String(http.StatusBadRequest,
				   err.Error())
	}
	if err := repo.InitRepo(name, user.Name); err != nil {
		return c.String(http.StatusBadRequest,
				   err.Error())
	}
	return redirect(c, "repo/" + name)
}

func AddGroup(c echo.Context, user db.User) error {
	name := c.QueryString()
	if err := user.CreateGroup(name); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return redirect(c, "groups/" + name)
}

func ChangePassword(c echo.Context, user db.User) error {
	passwd := c.QueryString()
	if err := user.ChangePassword(passwd); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return redirect(c, "")
}

func Disconnect(c echo.Context, user db.User) error {
	if err := user.Disconnect(); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	cookie := http.Cookie{
		Domain: config.Cfg.Web.Domain,
		Name: "auth_id",
		Expires: time.Unix(0, 0),
		Value: "",
		Path: "/",
	}
	c.SetCookie(&cookie)
	return c.Redirect(http.StatusFound, "/")
}

func DisconnectAll(c echo.Context, user db.User) error {
	if err := user.DisconnectAll(); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return redirect(c, "")
}
