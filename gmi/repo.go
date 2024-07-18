package gmi

import (
	"strconv"
	"strings"
	"net/http"
	"io"
	"bytes"

        "gemigit/db"
        "gemigit/repo"

	"github.com/labstack/echo/v4"
	"github.com/gabriel-vasile/mimetype"
)

func redirect(c echo.Context, after string) error {
	return c.Redirect(http.StatusFound, "/account/" + after)
}

func showFileContent(content string) string {
	lines := strings.Split(content, "\n")
	file := ""
	for i, line := range lines {
		file += strconv.Itoa(i) + "\t" + line + "\n"
	}
	return strings.Replace(file, "%", "%%", -1)
}

func serveFile(c echo.Context, name string, user string, file string) error {
	repofile, err := repo.GetFile(name, user, file)
	if err != nil { return err }
	reader, err := repofile.Reader()
	if err != nil { return err }
	var buf bytes.Buffer
	tee := io.TeeReader(reader, &buf)
	mime, err := mimetype.DetectReader(tee)
	if err != nil { return err }
	c.Response().Header().Add("Content-Type", mime.String())
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Write(buf.Bytes())
	io.Copy(c.Response().Writer, reader)
	return nil
}

// Private

func RepoFiles(c echo.Context, user db.User) error {
	query := c.QueryString()
	if query == "" {
		return showRepo(c, user, pageFiles, true)
	}
	repofile, err := repo.GetFile(c.Param("repo"), user.Name, query)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	contents, err := repofile.Contents()
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return c.HTML(http.StatusOK, contents)
}

func RepoFileContent(c echo.Context, user db.User) error {
	content, err := repo.GetPrivateFile(c.Param("repo"), user.Name,
				c.Param("blob"), user.Signature)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	header := "=>/account/repo/" + c.Param("repo") + "/files Go Back\n\n"
	return c.HTML(http.StatusOK, header + showFileContent(content))
}

func RepoFile(c echo.Context, user db.User) error {
	err := serveFile(c, c.Param("repo"), user.Name, c.Param("*"))
	if err != nil { return c.String(http.StatusBadRequest, err.Error()) }
	return nil
}

func TogglePublic(c echo.Context, user db.User) error {
	if err := user.TogglePublic(c.Param("repo"));
	   err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return redirect(c, "repo/" + c.Param("repo"))
}

func ChangeRepoName(c echo.Context, user db.User) error {
	newname := c.QueryString()
	// should check if repo exist and if the new name is free
	if err := repo.ChangeRepoDir(c.Param("repo"), user.Name, newname);
	   err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := user.ChangeRepoName(c.Param("repo"), newname);

	   err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return redirect(c, "repo/" + newname)
}

func ChangeRepoDesc(c echo.Context, user db.User) error {
	newdesc := c.QueryString()
	if err := user.ChangeRepoDesc(c.Param("repo"), newdesc); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return redirect(c, "repo/" + c.Param("repo"))
}

func DeleteRepo(c echo.Context, user db.User) error {
	name := c.QueryString()
	if name != c.Param("repo") {
		return redirect(c, "repo/" + c.Param("repo"))
	}
	// check if repo exist
	if err := repo.RemoveRepo(name, user.Name);
	   err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := user.DeleteRepo(name);
	   err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return redirect(c, "")
}

func RepoRefs(c echo.Context, user db.User) error {
	return showRepo(c, user, pageRefs, true)
}

func RepoLicense(c echo.Context, user db.User) error {
	return showRepo(c, user, pageLicense, true)
}

func RepoReadme(c echo.Context, user db.User) error {
	return showRepo(c, user, pageReadme, true)
}

func RepoLog(c echo.Context, user db.User) error {
	return showRepo(c, user, pageLog, true)
}
