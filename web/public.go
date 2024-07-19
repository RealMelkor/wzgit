package web

import (
	"gemigit/repo"
	"gemigit/db"
	"net/http"
	"github.com/labstack/echo/v4"
	"io"
)

func PublicFile(c echo.Context) error {
	err := serveFile(c, c.Param("repo"), c.Param("user"), c.Param("*"))
	if err != nil { return c.String(http.StatusBadRequest, err.Error()) }
	return nil
}

func PublicFileContent(c echo.Context) error {
	out, err := repo.GetPublicFile(
		c.Param("repo"), c.Param("user"), c.Param("blob"))
        if err != nil { return err }
	c.Response().WriteHeader(http.StatusOK)
	_, err = io.Copy(c.Response().Writer, out)
        return err
}

func PublicRefs(c echo.Context) error {
	return showRepo(c, db.User{}, pageRefs, false)
}

func PublicLicense(c echo.Context) error {
	return showRepo(c, db.User{}, pageLicense, false)
}

func PublicReadme(c echo.Context) error {
	return showRepo(c, db.User{}, pageReadme, false)
}

func PublicLog(c echo.Context) error {
	return showRepo(c, db.User{}, pageLog, false)
}

func PublicFiles(c echo.Context) error {
	return showRepo(c, db.User{}, pageFiles, false)
}
