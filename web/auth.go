package web

import (
	"crypto/rand"
	"errors"
	"encoding/base64"
	"net/http"
	"wzgit/auth"
	"wzgit/config"

	"github.com/labstack/echo/v4"
)

func token() (string, error) {
	var src [64]byte
	_, err := rand.Read(src[:])
	if err != nil { return "", err }
	return base64.StdEncoding.EncodeToString(src[:]), nil
}

func Register(c echo.Context) error {
	if err := captchaVerify(c); err != nil { return err }
	name := c.Request().PostFormValue("username")
	password := c.Request().PostFormValue("password")
	confirm := c.Request().PostFormValue("password_confirm")
	if confirm != password {
		return errors.New("passwords are not the same")
	}
	err := auth.Register(name, password, c.RealIP())
	if err != nil { return err }
	return render(c, "register_success.html", nil)
}

func Login(c echo.Context) error {
	if err := captchaVerify(c); err != nil { return err }
	name := c.Request().PostFormValue("username")
	pass := c.Request().PostFormValue("password")
	code := c.Request().PostFormValue("otp")
	sig, err := token()
	if err != nil { return err }
	err = auth.Connect(name, pass, sig, c.RealIP())
	if err != nil && err.Error() == "token required" {
		err = auth.LoginOTP(sig, code)
	}
	if err != nil { return err }
	cookie := http.Cookie{
		Domain: config.Cfg.Web.Domain,
		Name: "auth_id",
		Value: sig,
	}
	c.SetCookie(&cookie)
	return c.Redirect(http.StatusFound, "/" + name)
}
