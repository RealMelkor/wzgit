package web

import (
	"errors"
	"fmt"
	"gemigit/access"
	"gemigit/config"
	"gemigit/db"
	"gemigit/repo"
	"gemigit/csrf"
	"io"
	"log"
	"strconv"
	"strings"
	"html/template"
	"embed"
	"net/http"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/labstack/echo/v4"
)

//go:embed templates/*
var templatesFS embed.FS

const (
	pageLog = iota
	pageFiles
	pageRefs
	pageLicense
	pageReadme
)

var templates *template.Template

func LoadTemplate() error {
	var err error
	templates = template.New("html")
	t := templates.Funcs(template.FuncMap {
		"AccessFirst": accessFirstOption,
		"AccessSecond": accessSecondOption,
		"AccessPrivilege": privilegeToString,
	})
	_, err = t.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return err
	}
	log.Println("Templates loaded")
	return nil
}

func showRepoFile(user string, reponame string, file string) (string, error) {
        out, err := repo.GetFile(reponame, user, file)
        if err != nil {
                return "", err
        }
        reader, err := out.Reader()
        if err != nil {
                return "", err
        }
        buf, err := io.ReadAll(reader)
        if err != nil {
                return "", err
        }
        return string(buf), nil
}

func ShowIndex(c echo.Context, isConnected bool) (error) {
	data := struct {
		Title		string
		Registration	bool
		Connected	bool
		Public		bool
		Captcha		bool
	}{
		Title: config.Cfg.Title,
		Registration: config.Cfg.Users.Registration,
		Connected: isConnected,
		Public: isConnected || config.Cfg.Git.Public,
		Captcha: config.Cfg.Catpcha.Enabled,
	}
	return render(c, "index.html", data)
}

func ShowAccount(c echo.Context, user db.User) (error) {
	repoNames := []string{}
	repos, err := user.GetRepos(false)
	if err != nil {
		repoNames = []string{"Failed to load repositories"}
		log.Println(err)
	} else {
		for _, repo := range repos {
			repoNames = append(repoNames, repo.Name)
		}
	}
	accessRepos, err := user.HasReadAccessTo()
	sessions, err := user.GetSessionsCount()
	if err != nil {
		log.Println(err)
		return c.String(http.StatusBadRequest, "Unexpected error")
	}
	if sessions == 1 {
		sessions = 0
	}
	data := struct {
		User db.User
		Repositories []string
		RepositoriesAccess []db.Repo
		Sessions int
		CSRF string
	}{
		User: user,
		Repositories: repoNames,
		RepositoriesAccess: accessRepos,
		Sessions: sessions,
		CSRF: csrf.Token(user.Signature),
	}
	return render(c, "account.html", data)
}

func ShowGroups(c echo.Context, user db.User) (error) {
	groups, err := user.GetGroups()
	if err != nil { return err }
	data := struct {
		User	db.User
		Groups	[]db.Group
		CSRF	string
	}{
		User:	user,
		Groups:	groups,
		CSRF:	csrf.Token(user.Signature),
	}
	return render(c, "group_list.html", data)
}

func ShowMembers(c echo.Context, user db.User) (error) {
	group := c.Param("group")
	isOwner, err := user.IsInGroup(group)
	if err != nil {
		return c.String(http.StatusBadRequest,
				   "Group not found")
	}

	members, err := user.GetMembers(group)
	if err != nil {
		log.Println(err.Error())
		return c.String(http.StatusInternalServerError,
				   "Failed to fetch group members")
	}
	desc, err := db.GetGroupDesc(group)
	if err != nil {
		log.Println(err.Error())
		return c.String(http.StatusInternalServerError,
				   "Failed to fetch group description")
	}

	owner := ""
	if isOwner {
		owner = user.Name
	} else {
		m, err := db.GetGroupOwner(group)
		if err != nil {
			log.Println(err.Error())
			return c.String(http.StatusBadRequest,
					   "Failed to fetch group owner")
		}
		owner = m.Name
	}

	data := struct {
		Members []db.Member
		MembersCount int
		IsOwner bool
		Owner string
		Group string
		Description string
		CSRF string
	}{
		Members: members,
		MembersCount: len(members),
		IsOwner: isOwner,
		Owner: owner,
		Group: group,
		Description: desc,
		CSRF: csrf.Token(user.Signature),
	}
	return render(c, "group.html", data)
}

func getRepo(c echo.Context, user db.User, owner bool) (string, string, error) {
        username := ""
        if owner {
                username = user.Name
        } else {
                username = c.Param("user")
                ret, err := db.IsRepoPublic(c.Param("repo"), c.Param("user"))
		if !ret {
			err := access.HasReadAccess(c.Param("repo"),
						    c.Param("user"),
						    user.Name)
			ret = err == nil
		}
                if !ret || err != nil {
                        return "", "", c.String(http.StatusBadRequest,
				"No repository called " + c.Param("repo") +
                                " by user " + c.Param("user"))
                }
        }
	return username, c.Param("repo"), nil
}

func hasFile(name string, author string, file string) bool {
	ret, err := repo.GetFile(name, author, file)
	if ret != nil && err == nil {
		return true
	}
	return false
}

type commit struct {
	Message	string
	Info	string
}

type file struct {
	Hash string
	Info string
}

type branch struct {
	Name string
	Info string
}

func showRepoLogs(name string, author string) (any, error) {
	ret, err := repo.GetCommits(name, author)
	if ret == nil || err == transport.ErrEmptyRemoteRepository {
		return nil, nil
	}
	if err != nil {
		log.Println(err.Error())
		return nil, errors.New("Corrupted repository")
	}
	commits := []commit{}
	maximum := config.Cfg.Git.MaximumCommits
	for i := 0; maximum == 0 || i < maximum; i++ {
		c, err := ret.Next()
		if err != nil {
			if err.Error() == "EOF" { break }
			log.Println(err.Error())
			return nil, err
		}
		info := c.Hash.String() + ", by " + c.Author.Name + " on " +
			c.Author.When.Format("2006-01-02 15:04:05")
		commits = append(commits, commit{Info: info,
						 Message: c.Message})
	}
	return commits, nil
}

func showRepoFiles(name string, author string) (any, error) {
	ret, err := repo.GetFiles(name, author)
	if ret == nil || err == transport.ErrEmptyRemoteRepository {
		return nil, nil
	}
	if err != nil {
		log.Println(err.Error())
		return nil, errors.New("Corrupted repository")
	}
	files := []file{}
	err = ret.ForEach(func(f *object.File) error {
		info := f.Mode.String() + " " + f.Name +
			" " + strconv.Itoa(int(f.Size))
		files = append(files, file{Info: info,
					   Hash: f.Blob.Hash.String()})
		return nil
	})
	return files, nil
}

func showRepoRefs(name string, author string) (any, error) {
	refs, err := repo.GetRefs(name, author)
	if refs == nil || err == transport.ErrEmptyRemoteRepository {
		return nil, nil
	}
	if err != nil {
		log.Println(err)
		return nil, errors.New("Corrupted repository")
	}
	branches := []branch{}
	tags := []branch{}
	err = refs.ForEach(func(c *plumbing.Reference) error {
		if c.Type().String() != "hash-reference" ||
		   c.Name().IsRemote() {
			return nil
		}
		var b branch
		b.Name = c.Name().String()
		b.Name = b.Name[strings.LastIndex(b.Name, "/") + 1:]
		b.Info = "last commit on "

		commit, err := repo.GetCommit(name, author, c.Hash())
		if err != nil {
			b.Info = "failed to fetch commit"
		} else {
			when := commit.Author.When
			str := fmt.Sprintf(
				"%d-%02d-%02d %02d:%02d:%02d",
				when.Year(), int(when.Month()),
				when.Day(), when.Hour(),
				when.Minute(), when.Second())
			b.Info += str + " by " + commit.Author.Name
		}
		if c.Name().IsBranch() {
			branches = append(branches, b)
		} else {
			tags = append(tags, b)
		}
		return nil
	})
	refs.Close()
	data := struct {
		Branches []branch
		Tags []branch
	}{
		branches,
		tags,
	}
	return data, nil
}

func showRepoLicense(name string, author string) (string, error) {
	content, err := showRepoFile(author, name, "LICENSE")
	if err != nil {
		return "", errors.New("No license found")
	}
	return content, nil
}

func showRepoReadme(name string, author string) (any, error) {
	content, err := showRepoFile(author, name, "README.md")
	if err != nil { return "", errors.New("No readme found") }
	return template.HTML(mdToHTML([]byte(content))), nil
}

func showRepo(c echo.Context, user db.User, page int, owner bool) (error) {
	loggedAs := ""
	author, name, err := getRepo(c, user, owner)
	if err != nil {
		log.Println(err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}
	desc, err := db.GetRepoDesc(name, author)
	if err != nil {
		log.Println(err.Error())
		return c.String(http.StatusBadRequest, "Repository not found")
	}
	protocol := "http"
	if config.Cfg.Git.Http.Https {
		protocol = "https"
	}
	public, err := db.IsRepoPublic(name, author)
	if err != nil {
		log.Println(err.Error())
		return c.String(http.StatusBadRequest, "Repository not found")
	}
	if public && config.Cfg.Git.Public {
		loggedAs = "anon@"
	}

	var content any
	contentType := ""
	switch page {
	case pageLog:
		content, err = showRepoLogs(name, author)
		contentType = "log"
	case pageFiles:
		content, err = showRepoFiles(name, author)
		contentType = "files"
	case pageRefs:
		content, err = showRepoRefs(name, author)
		contentType = "refs"
	case pageLicense:
		content, err = showRepoLicense(name, author)
		contentType = "license"
	case pageReadme:
		content, err = showRepoReadme(name, author)
		contentType = "readme"
	}
	if err != nil {
		return c.String(http.StatusBadRequest,
				   "Invalid repository")
	}

	data := struct {
		HasHTTP bool
		HttpProtocol string
		HttpDomain string
		HasSSH bool
		SshDomain string
		LoggedAs string
		User string
		Description string
		Repo string
		Public bool
		HasReadme bool
		HasLicense bool
		Content any
		CSRF string
		Page string
	}{
		HasHTTP: config.Cfg.Git.Http.Enabled,
		HttpProtocol: protocol,
		HttpDomain: config.Cfg.Git.Http.Domain,
		HasSSH: config.Cfg.Git.SSH.Enabled,
		SshDomain: config.Cfg.Git.SSH.Domain,
		LoggedAs: loggedAs,
		User: author,
		Description: desc,
		Repo: name,
		Public: public,
		HasReadme: hasFile(name, author, "README.html") ||
			   hasFile(name, author, "README.md") ||
			   hasFile(name, author, "README"),
		HasLicense: hasFile(name, author, "LICENSE"),
		Content: content,
		CSRF: csrf.Token(user.Signature),
		Page: contentType,
	}
	if owner {
		return render(c, "repo.html", data)
	}
	return render(c, "public_repo.html", data)
}

func PublicList(c echo.Context) (error) {
	repos, err := db.GetPublicRepo()
	if err != nil {
		log.Println(err.Error())
		return c.String(http.StatusInternalServerError,
				   "Internal error, " + err.Error())
	}
	return render(c, "public_list.html", repos)
}

func PublicAccount(c echo.Context) error {
	user, err := db.GetPublicUser(c.Param("user"))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	repos, err := user.GetRepos(true)
	if err != nil {
		return c.Redirect(http.StatusFound,
				   "Invalid account, " + err.Error())
	}
	data := struct {
		Name string
		Description string
		Repositories []db.Repo
	}{
		user.Name,
		user.Description,
		repos,
	}
	return render(c, "public_user.html", data)
}

func ShowAccess(c echo.Context, user db.User) error {
	repo, err := user.GetRepo(c.Param("repo"))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	access, err := db.GetRepoUserAccess(repo.ID)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	groups, err := db.GetRepoGroupAccess(repo.ID)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	data := struct {
		Repo string
		Collaborators []db.Access
		Groups []db.Access
		Owner bool
		CSRF string
	}{
		Repo: repo.Name,
		Collaborators: access,
		Groups: groups,
		Owner: true,
		CSRF: csrf.Token(user.Signature),
	}
	return render(c, "repo_access.html", data)
}

func ShowOTP(c echo.Context, user db.User) error {
	data := struct {
		Secret bool
		CSRF string
	}{
		Secret: user.Secret != "",
		CSRF: csrf.Token(user.Signature),
	}
	return render(c, "otp.html", data)
}

func ShowTokens(c echo.Context, user db.User) error {

	tokens, err := user.GetTokens()
	if err != nil { return err }

	data := struct {
		User	db.User
		Tokens	[]db.Token
		Secure	bool
		CSRF	string
	}{
		User:	user,
		Tokens:	tokens,
		Secure:	user.SecureGit,
		CSRF:	csrf.Token(user.Signature),
	}
	return render(c, "token.html", data)
}
