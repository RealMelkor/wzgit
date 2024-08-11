package web

import (
	"strconv"
	"strings"
	"net/http"
	"io"
	"errors"
	"bytes"

	"wzgit/db"
	"wzgit/repo"

	"github.com/labstack/echo/v4"
	"github.com/gabriel-vasile/mimetype"
)

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
		return showRepo(c, user, pageFiles)
	}
	repofile, err := repo.GetFile(c.Param("repo"), user.Name, query)
	if err != nil { return err }
	contents, err := repofile.Contents()
	if err != nil { return err }
	return c.HTML(http.StatusOK, contents)
}

func RepoFileContent(c echo.Context, user db.User) error {
	out, err := repo.GetPrivateFile(c.Param("repo"), user.Name,
				c.Param("blob"), user.Signature)
        if err != nil { return err }
	defer out.Close()
        c.Response().WriteHeader(http.StatusOK)
        _, err = io.Copy(c.Response().Writer, out)
        return err
}

func RepoFile(c echo.Context, user db.User) error {
	return serveFile(c, c.Param("repo"), user.Name, c.Param("*"))
}

func TogglePublic(c echo.Context, user db.User) error {
	if err := user.TogglePublic(c.Param("repo")); err != nil { return err }
	return redirect(c, user, c.Param("repo"))
}

func ChangeRepoName(c echo.Context, user db.User) error {
	oldname := c.Param("repo")
	newname := c.Request().PostFormValue("name")
	if err := db.IsRepoNameValid(newname); err != nil { return err }
	if id, _ := db.GetRepoID(newname, user.ID); id != -1 {
		return errors.New(
			"One of your repositories is already named " + newname)
	}
	err := user.ChangeRepoName(oldname, newname)
	if err != nil { return err }
	err = repo.ChangeRepoDir(oldname, user.Name, newname)
	if err != nil {
		user.ChangeRepoName(newname, oldname)
		return err
	}
	return redirect(c, user, newname)
}

func ChangeRepoDesc(c echo.Context, user db.User) error {
	newdesc := c.Request().PostFormValue("desc")
	if err := user.ChangeRepoDesc(c.Param("repo"), newdesc); err != nil {
		return err
	}
	return redirect(c, user, c.Param("repo"))
}

func DeleteRepo(c echo.Context, user db.User) error {
	name := c.QueryString()
	if name != c.Param("repo") {
		user.Set("repo_delete_confirm", c.Param("repo"))
		return redirect(c, user, c.Param("repo"))
	}
	// check if repo exist
	if err := repo.RemoveRepo(name, user.Name); err != nil { return err }
	if err := user.DeleteRepo(name); err != nil { return err }
	return redirect(c, user, "")
}

func RepoRefs(c echo.Context, user db.User) error {
	return showRepo(c, user, pageRefs)
}

func RepoLicense(c echo.Context, user db.User) error {
	return showRepo(c, user, pageLicense)
}

func RepoReadme(c echo.Context, user db.User) error {
	return showRepo(c, user, pageReadme)
}

func RepoLog(c echo.Context, user db.User) error {
	return showRepo(c, user, pageLog)
}
