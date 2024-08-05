package web

import (
	"github.com/labstack/echo/v4"
	"github.com/dchest/captcha"
	"errors"
	"net/http"

	"wzgit/config"
)

func captchaNew() string {
	if !config.Cfg.Captcha.Enabled { return "" }
	return captcha.NewLen(config.Cfg.Captcha.Length)
}

func captchaImage(c echo.Context) error {
	id := c.QueryString()
	if id == "" { return errors.New("no captcha") }
	c.Response().WriteHeader(http.StatusOK)
	return captcha.WriteImage(c.Response().Writer, id,
		captcha.StdWidth, captcha.StdHeight)
}

func captchaVerify(c echo.Context) error {
	if !config.Cfg.Captcha.Enabled { return nil }
	id := c.Request().PostFormValue("captcha_id")
	answer := c.Request().PostFormValue("captcha")
	if !captcha.VerifyString(id, answer) {
		return errors.New("invalid captcha")
	}
	return nil
}
