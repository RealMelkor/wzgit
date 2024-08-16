package web

import (
	"errors"
	"fmt"
	"wzgit/access"
	"wzgit/config"
	"wzgit/db"
	"wzgit/repo"
	"wzgit/csrf"
	"io"
	"log"
	"strconv"
	"strings"
	"html/template"
	"embed"

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
		"sub": func(y, x int) int {
			return x - y
		},
		"title": func(s string) string {
			if len(s) < 2 { return s }
			return strings.ToUpper(s[0:1]) + s[1:]
		},
		"captcha": captchaNew,
	})
	_, err = t.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return err
	}
	log.Println("Templates loaded")
	return nil
}

func getRepoFile(user string, reponame string, file string) (io.Reader, error) {
        out, err := repo.GetFile(reponame, user, file)
        if err != nil { return nil, err }
        return out.Reader()
}

func showRepoFile(user string, reponame string, file string) (string, error) {
        reader, err := getRepoFile(user, reponame, file)
        if err != nil { return "", err }
        buf, err := io.ReadAll(reader)
        if err != nil { return "", err }
        return string(buf), nil
}

func showIndex(c echo.Context, isConnected bool, err string) (error) {
	data := struct {
		Title		string
		Registration	bool
		Connected	bool
		Public		bool
		Captcha		bool
		Error		string
	}{
		Title:		config.Cfg.Title,
		Registration:	config.Cfg.Users.Registration,
		Connected:	isConnected,
		Public:		isConnected || config.Cfg.Git.Public,
		Captcha:	config.Cfg.Captcha.Enabled,
		Error:		err,
	}
	return render(c, "index.html", data)
}

func ShowIndex(c echo.Context, isConnected bool) (error) {
	return showIndex(c, isConnected, "")
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
	if err != nil { return err }
	if sessions == 1 {
		sessions = 0
	}
	data := struct {
		User db.User
		Repositories []string
		RepositoriesAccess []db.Repo
		Sessions int
		CSRF string
		LDAP bool
	}{
		User: user,
		Repositories: repoNames,
		RepositoriesAccess: accessRepos,
		Sessions: sessions,
		CSRF: csrf.Token(user.Signature),
		LDAP: config.Cfg.Ldap.Enabled,
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
	if err != nil { return err }
	members, err := user.GetMembers(group)
	if err != nil { return err }
	desc, err := db.GetGroupDesc(group)
	if err != nil { return err }

	owner := ""
	if isOwner {
		owner = user.Name
	} else {
		m, err := db.GetGroupOwner(group)
		if err != nil { return err }
		owner = m.Name
	}

	data := struct {
		User	db.User
		Members []db.Member
		MembersCount int
		IsOwner bool
		Owner string
		Group string
		Description string
		CSRF string
	}{
		User:		user,
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

func getRepo(c echo.Context, user db.User) (string, string, error) {
	username := c.Param("user")
	repo, err := db.GetRepo(c.Param("repo"), c.Param("user"))
	if err == nil && !repo.IsPublic {
		err = access.HasReadAccess(c.Param("repo"), c.Param("user"),
					    user.Name)
	}
	if err != nil { return "", "", err }
	return username, repo.Name, nil
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
	if err != nil { return nil, err }
	commits := []commit{}
	maximum := config.Cfg.Git.MaximumCommits
	for i := 0; maximum == 0 || i < maximum; i++ {
		c, err := ret.Next()
		if err != nil {
			if err.Error() == "EOF" { break }
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
	if err != nil { return nil, err }
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
	if err != nil { return nil, err }
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
	if err != nil { return "", errors.New("No license found") }
	return content, nil
}

func showRepoReadme(name string, author string, w io.Writer) error {
	content, err := showRepoFile(author, name, "README.md")
	if err != nil {
		r, err := getRepoFile(author, name, "README")
		if err != nil { return errors.New("No readme found") }
		return textToHTML(r, w)
	}
	return readmeMarkdown([]byte(content), w)
}

func showRepo(c echo.Context, user db.User, page int) (error) {
	loggedAs := ""
	author, name, err := getRepo(c, user)
	if err != nil { return err }
	desc, err := db.GetRepoDesc(name, author)
	if err != nil { return err }
	protocol := "http"
	if config.Cfg.Git.Http.Https { protocol = "https" }
	public, err := db.IsRepoPublic(name, author)
	if err != nil { return err }
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
		content = true
		contentType = "readme"
	}
	if err != nil { return err }

	data := struct {
		User db.User
		HasHTTP bool
		HttpProtocol string
		HttpDomain string
		HasSSH bool
		SshDomain string
		LoggedAs string
		Author string
		Description string
		Repo string
		Public bool
		Owner bool
		HasReadme bool
		HasLicense bool
		Content any
		CSRF string
	}{
		User: user,
		HasHTTP: config.Cfg.Git.Http.Enabled,
		HttpProtocol: protocol,
		HttpDomain: config.Cfg.Git.Http.Domain,
		HasSSH: config.Cfg.Git.SSH.Enabled,
		SshDomain: config.Cfg.Git.SSH.Domain,
		LoggedAs: loggedAs,
		Author: author,
		Description: desc,
		Repo: name,
		Public: public,
		Owner: author == user.Name,
		HasReadme: hasFile(name, author, "README.md") ||
			   hasFile(name, author, "README"),
		HasLicense: hasFile(name, author, "LICENSE"),
		Content: content != nil,
		CSRF: csrf.Token(user.Signature),
	}
	f := func(w io.Writer) error {
		if contentType == "readme" {
			return showRepoReadme(name, author, w)
		}
		return templates.Lookup(contentType).Execute(w, content)
	}
	return renderCustom(c, "repo.html", data, f)
}

func PublicList(c echo.Context) (error) {
	repos, err := db.GetPublicRepo()
	if err != nil { return err }
	return render(c, "public_list.html", repos)
}

func PublicAccount(c echo.Context) error {
	user, err := db.GetPublicUser(c.Param("user"))
	if err != nil { return err }
	u, _ := getUser(c)
	repos, err := user.GetRepos(u.ID != user.ID)
	if err != nil { return err }
	if user.ID == u.ID {
		accessRepos, err := user.HasReadAccessTo()
		if err != nil { return err }
		repos = append(repos, accessRepos...)
	}
	data := struct {
		User db.User
		Name string
		Description string
		Repositories []db.Repo
		CSRF string
	}{
		User: u,
		Name: user.Name,
		Description: user.Description,
		Repositories: repos,
		CSRF: csrf.Token(user.Signature),
	}
	return render(c, "user.html", data)
}

func ShowAccess(c echo.Context, user db.User) error {
	repo, err := user.GetRepo(c.Param("repo"))
	if err != nil { return err }
	access, err := db.GetRepoUserAccess(repo.ID)
	if err != nil { return err }
	groups, err := db.GetRepoGroupAccess(repo.ID)
	if err != nil { return err }
	data := struct {
		User db.User
		Repo string
		Collaborators []db.Access
		Groups []db.Access
		Owner bool
		CSRF string
	}{
		User: user,
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
		User db.User
		Secret bool
		CSRF string
	}{
		User: user,
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

func ShowPasswd(c echo.Context, user db.User) error {

	tokens, err := user.GetTokens()
	if err != nil { return err }

	data := struct {
		User	db.User
		Tokens	[]db.Token
		CSRF	string
	}{
		User:	user,
		Tokens:	tokens,
		CSRF:	csrf.Token(user.Signature),
	}
	return render(c, "passwd.html", data)
}
