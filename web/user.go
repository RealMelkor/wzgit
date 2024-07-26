package web

import (
	"time"
	"net/http"
	"errors"
        "gemigit/db"
        "gemigit/repo"
        "gemigit/config"

	"github.com/labstack/echo/v4"
)

func ChangeDesc(c echo.Context, user db.User) error {
	desc := c.Request().PostFormValue("description")
	if err := user.ChangeDescription(desc); err != nil { return err }
	return redirect(c, "")
}

func AddRepo(c echo.Context, user db.User) error {
	name := c.Request().PostFormValue("repo")
	if err := user.CreateRepo(name); err != nil { return err }
	if err := repo.InitRepo(name, user.Name); err != nil { return err }
	return redirect(c, "/repo/" + name)
}

func ChangePassword(c echo.Context, user db.User) error {
	oldPass := c.Request().PostFormValue("old_password")
	newPass := c.Request().PostFormValue("new_password")
	confirm := c.Request().PostFormValue("confirm")
	if newPass != confirm { return errors.New("Passwords don't match") }
	if err := db.CheckAuth(user.Name, oldPass); err != nil { return err }
	if err := user.ChangePassword(newPass); err != nil { return err }
	return redirect(c, "")
}

func Disconnect(c echo.Context, user db.User) error {
	if err := user.Disconnect(); err != nil { return err }
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
	if err := user.DisconnectAll(); err != nil { return err }
	return redirect(c, "")
}
