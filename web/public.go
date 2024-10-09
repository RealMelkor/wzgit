package web

import (
	"wzgit/repo"
	"net/http"
	"github.com/labstack/echo/v4"
	"io"
)

func PublicFile(c echo.Context) error {
	return serveFile(c, c.Param("repo"), c.Param("user"), c.Param("*"))
}

func PublicFileContent(c echo.Context) error {
	sig, _ := c.Cookie("auth_id")
	out, err := repo.GetPublicFile(
		c.Param("repo"), c.Param("user"), c.Param("blob"), sig.Value)
	if err != nil { return err }
	defer out.Close()
	c.Response().WriteHeader(http.StatusOK)
	_, err = io.Copy(c.Response().Writer, out)
	return err
}

func PublicRefs(c echo.Context) error {
	user, _ := getUser(c)
	return showRepo(c, user, pageRefs)
}

func PublicLicense(c echo.Context) error {
	user, _ := getUser(c)
	return showRepo(c, user, pageLicense)
}

func PublicReadme(c echo.Context) error {
	user, _ := getUser(c)
	return showRepo(c, user, pageReadme)
}

func PublicLog(c echo.Context) error {
	user, _ := getUser(c)
	return showRepo(c, user, pageLog)
}

func PublicFiles(c echo.Context) error {
	user, _ := getUser(c)
	return showRepo(c, user, pageFiles)
}
