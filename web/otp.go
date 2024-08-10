package web

import (
	"wzgit/db"
	"wzgit/auth"

	"github.com/labstack/echo/v4"

	"bytes"
	"image/png"
	"net/http"
)

var keys = make(map[string]string)
func otpRedirect(c echo.Context) error {
	return c.Redirect(http.StatusFound, "/account/otp")
}

func CreateTOTP(c echo.Context, user db.User) error {

	key, err := auth.GenerateOTP(user.Name)
	if err != nil { return err }

	var buf bytes.Buffer
	img, err := key.Image(200, 200)
	if err != nil { return err }
	png.Encode(&buf, img)

	keys[user.Signature] = key.Secret()

	return c.Blob(http.StatusOK, "image/png", buf.Bytes())
}

func ConfirmTOTP(c echo.Context, user db.User) error {

	code := c.Request().PostFormValue("code")
	key, exist := keys[user.Signature]
	if exist {
		if err := auth.CheckOTP(key, code); err != nil { return err }
	}
	if err := user.SetSecret(key); err != nil { return err }

	return otpRedirect(c)
}

func RemoveTOTP(c echo.Context, user db.User) error {
	code := c.Request().PostFormValue("code")
	if err := auth.CheckOTP(user.Secret, code); err != nil { return err }
	if err := user.SetSecret(""); err != nil { return err }
	return otpRedirect(c)
}
