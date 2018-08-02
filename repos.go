package main

import (
	"log"
	"os"
	"path"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

type Repositories struct {
	Cache string
}

func NewRepositories() (r *Repositories, err error) {
	home := os.Getenv("HOME")

	r = &Repositories{
		Cache: path.Join(home, ".simple-deps"),
	}

	err = os.MkdirAll(r.Cache, 0755)
	if err != nil {
		return nil, err
	}

	return
}

func (repos *Repositories) GetRepositoryHashRecursively(p string) (h plumbing.Hash, err error) {
	for {
		r, err := git.PlainOpen(p)
		if err != nil {
			p = path.Dir(p)
			if p == "." || p == "/" {
				return plumbing.ZeroHash, err
			}
			continue
		}

		ref, err := r.Head()
		if err != nil {
			return plumbing.ZeroHash, err
		}

		h = ref.Hash()

		break
	}

	return
}

func (repos *Repositories) GetRepositoryHash(p string) (h plumbing.Hash, err error) {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return plumbing.ZeroHash, nil
	}

	r, err := git.PlainOpen(p)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	ref, err := r.Head()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	h = ref.Hash()

	return
}

func (repos *Repositories) UpdateRepository(source, path string, pull, fetch bool) (*git.Repository, error) {
	pullNecessary := true

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Cloning %s %s", source, path)

		_, err := git.PlainClone(path, false, &git.CloneOptions{
			URL:      source,
			Progress: os.Stdout,
		})
		if err != nil {
			return nil, err
		}

		pullNecessary = false
	}

	r, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	if pullNecessary {
		if pull {
			log.Printf("Pull %s", path)
			err = w.Pull(&git.PullOptions{
				RemoteName: "origin",
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return nil, err
			}
		} else if fetch {
			log.Printf("Fetch %s", path)
			err = r.Fetch(&git.FetchOptions{
				RemoteName: "origin",
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return nil, err
			}
		}
	}

	return r, nil
}

func (repos *Repositories) HasCommit(path string, version string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	r, err := git.PlainOpen(path)
	if err != nil {
		return false
	}

	commit, err := r.CommitObject(plumbing.NewHash(version))
	if err != nil {
		return false
	}

	if false {
		log.Printf("%v", commit)
	}

	return true
}

func (repos *Repositories) CloneDependency(lib *Library, directory string, useHead bool) (clonePath string, err error) {
	name := path.Base(lib.URL.Path)
	name = strings.TrimSuffix(name, path.Ext(name))
	cached := path.Join(repos.Cache, name)
	p := path.Join(directory, name)

	pullCache := useHead
	if lib.Version == "*" || !repos.HasCommit(cached, lib.Version) {
		log.Printf("Version mismatch, pulling")
		pullCache = true
	}
	if !pullCache {
		log.Printf("Cache looks good")
	}

	_, err = repos.UpdateRepository(lib.URL.String(), cached, pullCache, false)
	if err != nil {
		return "", err
	}

	r, err := repos.UpdateRepository(cached, p, useHead, true)
	if err != nil {
		return "", err
	}

	wc, err := r.Worktree()
	if err != nil {
		return "", err
	}

	ref, err := r.Head()
	if err != nil {
		return "", err
	}

	head := ref.Hash().String()

	if useHead {
		if lib.Version != head {
			log.Printf("%s: Version changed: %v", name, head)
			lib.Version = head
			lib.Modified = true
		}
	}

	if lib.Version != "" {
		log.Printf("Checkout out %s (head = %s)", lib.Version, head)
		err = wc.Checkout(&git.CheckoutOptions{
			Hash:  plumbing.NewHash(lib.Version),
			Force: true,
		})
		if err != nil {
			return "", err
		}
	}

	return p, nil
}
