package gmi

import (
	"crypto/rand"
	"errors"
	"encoding/base64"

	"gemigit/auth"
	"net/http"
	"github.com/labstack/echo/v4"
)

func token() (string, error) {
	var src [64]byte
	_, err := rand.Read(src[:])
	if err != nil { return "", err }
	return base64.StdEncoding.EncodeToString(src[:]), nil
}

func Register(c echo.Context) error {
	name := c.Request().PostFormValue("username")
	password := c.Request().PostFormValue("password")
	confirm := c.Request().PostFormValue("password_confirm")
	if confirm != password {
		return errors.New("passwords are not the same")
	}
	err := auth.Register(name, password, c.RealIP())
	if err != nil { return err }
	data, err := execTemplate("register_success.gmi", nil)
	if err != nil { return err }
	return c.HTML(http.StatusOK, data)
}

func Login(c echo.Context) error {
	name := c.Request().PostFormValue("username")
	pass := c.Request().PostFormValue("password")
	sig, err := token()
	if err != nil { return err }
	err = auth.Connect(name, pass, sig, c.RealIP())
	if err != nil && err.Error() == "token required" {
		return c.Redirect(http.StatusFound, "/otp")
	}
	if err != nil { return c.String(http.StatusBadRequest, err.Error()) }
	cookie := http.Cookie{
		Domain: "127.0.0.1",
		Name: "auth_id",
		Value: sig,
	}
	c.SetCookie(&cookie)
	return c.Redirect(http.StatusFound, "/account")
}
