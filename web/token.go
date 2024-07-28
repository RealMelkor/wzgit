package web

import (
	"strconv"
	"errors"

	"gemigit/db"

	"github.com/labstack/echo/v4"
)

func CreateToken(c echo.Context, user db.User, readOnly bool) error {
	token, err := user.CreateToken(readOnly)
	if err != nil { return err }
	user.Set("new_token", token)
	return redirect(c, "/token")
}

func CreateWriteToken(c echo.Context, user db.User) error {
	return CreateToken(c, user, false)
}

func CreateReadToken(c echo.Context, user db.User) error {
	return CreateToken(c, user, true)
}

func ToggleTokenAuth(c echo.Context, user db.User) error {
	if err := user.ToggleSecure(); err != nil { return err }
	return redirect(c, "/token")
}

func RenewToken(c echo.Context, user db.User) error {
	id, err := strconv.Atoi(c.Param("token"))
	if err != nil || user.RenewToken(id) != nil {
		return errors.New("Invalid token")
	}
	return redirect(c, "/token")
}

func DeleteToken(c echo.Context, user db.User) error {
	id, err := strconv.Atoi(c.Param("token"))
	if err != nil || user.DeleteToken(id) != nil {
		return errors.New("Invalid token")
	}
	return redirect(c, "/token")
}
