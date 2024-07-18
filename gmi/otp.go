package gmi

import (
	"gemigit/auth"
	"gemigit/config"
	"gemigit/db"

	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"

	"log"
	"bytes"
	"image/png"
	"net/http"
)

var keys = make(map[string]string)

func otpRedirect(c echo.Context) error {
	return c.String(http.StatusFound, "/account/otp")
}

func CreateTOTP(c echo.Context, user db.User) error {

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer: config.Cfg.Title,
		AccountName: user.Name,
	})

	if err != nil {
		log.Println(err)
		return c.String(http.StatusBadRequest, "Unexpected error")
	}

	var buf bytes.Buffer
	img, err_ := key.Image(200, 200)
	if err_ != nil {
		log.Println(err)
		return c.String(http.StatusBadRequest, "Unexpected error")
	}
	png.Encode(&buf, img)

	keys[user.Signature] = key.Secret()

	return c.Blob(http.StatusOK, "image/png", buf.Bytes())
}

func ConfirmTOTP(c echo.Context, user db.User) error {

	query := c.QueryString()
	key, exist := keys[user.Signature]

	valid := false
	if exist { valid = totp.Validate(query, key) }
	if !valid {
		return c.String(http.StatusBadRequest, "Invalid code")
	}

	if err := user.SetSecret(key); err != nil {
		log.Println(err)
		return c.String(http.StatusBadRequest, "Unexpected error")
	}

	return otpRedirect(c)
}

func LoginOTP(c echo.Context, user db.User) error {

	query := c.QueryString()
	err := auth.LoginOTP(user.Signature, query)
	if err != nil && err.Error() == "wrong code" {
		return c.Redirect(http.StatusFound, "/otp")
	}
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.Redirect(http.StatusFound, "/account/")
}

func RemoveTOTP(c echo.Context, user db.User) error {
	query := c.QueryString()
	valid := totp.Validate(query, user.Secret)
	if !valid {
		return c.String(http.StatusFound, "/otp")
	}

	if err := user.SetSecret(""); err != nil {
		log.Println(err)
		return c.String(http.StatusBadRequest, "Unexpected error")
	}

	return otpRedirect(c)
}
