package repo

import (
	"errors"
	"gemigit/db"
	"io"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
)

var rootPath string
type repo struct {
	memory storage.Storer
	repository *git.Repository
	update time.Time
}
var repositories = make(map[string]repo)

func Init(path string) error {
	rootPath = path
	return os.MkdirAll(path, 0700)
}

func InitRepo(name string, username string) error {
	_, err := git.PlainInit(rootPath + "/" + username + "/" + name, true)
	return err
}

func RemoveRepo(name string, username string) error {
	return os.RemoveAll(rootPath + "/" + username + "/" + name)
}

func getRepo(name string, username string) (*git.Repository, error) {
	return git.PlainOpen(rootPath + "/" + username + "/" + name)
}

func GetCommit(name string, username string,
	       hash plumbing.Hash) (*object.Commit, error) {
	repo, err := getRepo(name, username)
	if err != nil {
		return nil, err
	}
	obj, err := repo.CommitObject(hash)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func GetCommits(name string, username string) (object.CommitIter, error) {
	repo, err := getRepo(name, username)
	if repo == nil || err != nil {
		return nil, err
	}
	ref, err := repo.Head()
	if err != nil {
		return nil, nil // Empty repo
	}
	cIter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, err
	}
	return cIter, nil
}

func GetRefs(name string, username string) (storer.ReferenceIter, error) {
	repo, err := getRepo(name, username)
	if repo == nil || err != nil {
		return nil, err
	}
	_, err = repo.Head()
	if err != nil {
		return nil, nil // Empty repo
	}
	refs, err := repo.References()
	if err != nil {
		return nil, err
	}
	return refs, nil
}

func getTree(name string, username string) (*object.Tree, error) {
	repo, err := getRepo(name, username)
	if repo == nil || err != nil {
		return nil, err
	}
	ref, err := repo.Head()
	if err != nil {
		return nil, nil // Empty repo
	}
	last, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}
	tree, err := repo.TreeObject(last.TreeHash)
	if err != nil {
		return nil, err
	}
	return tree, nil
}

func GetFiles(name string, username string) (*object.FileIter, error) {
	tree, err := getTree(name, username)
	if err != nil {
		return nil, err
	}
	if tree == nil {
		return nil, nil
	}
	return tree.Files(), nil
}

func GetFile(name string, username string, file string) (*object.File, error) {
	tree, err := getTree(name, username)
	if err != nil { return nil, err }
	if tree == nil { return nil, nil }
	out, err := tree.File(file)
	if err != nil { return nil, err }
	return out, nil
}

func GetPublicFile(name string, username string,
			hash string) (io.ReadCloser, error) {
	public, err := db.IsRepoPublic(name, username)
	if err != nil { return nil, err }
	if !public { return nil, errors.New("repository is private") }
	repo, err := git.PlainOpen(rootPath + "/" + username + "/" + name)
	if err != nil { return nil, err }
	file, err := repo.BlobObject(plumbing.NewHash(hash))
	if err != nil { return nil, err }
	reader, err := file.Reader()
	if err != nil { return nil, err }
	return reader, nil
}

func GetPrivateFile(name string, username string,
		    hash string, sig string) (io.ReadCloser, error) {
	user, b := db.GetUser(sig)
	if !b || username != user.Name {
		return nil, errors.New("invalid signature")
	}
	repo, err := git.PlainOpen(rootPath + "/" + username + "/" + name)
	if err != nil { return nil, err }
	file, err := repo.BlobObject(plumbing.NewHash(hash))
	if err != nil { return nil, err }
	reader, err := file.Reader()
	if err != nil { return nil, err }
	return reader, nil
}

func ChangeRepoDir(name string, username string, newname string) error {
	return os.Rename(rootPath + "/" + username + "/" + name,
			 rootPath + "/" + username + "/" + newname)
}
