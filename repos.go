package main

import (
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

type Repositories struct {
	NestedLayout bool
	Cache        string
}

func NewRepositories(nestedLayout bool) (r *Repositories, err error) {
	home := os.Getenv("HOME")

	r = &Repositories{
		NestedLayout: nestedLayout,
		Cache:        path.Join(home, ".simple-deps"),
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

func (repos *Repositories) UpdateRepository(name, source, path string, pull, fetch bool) (*git.Repository, error) {
	pullNecessary := true

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("%s: Cloning %s %s", name, source, path)

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
			log.Printf("%s: Pull %s", name, path)
			err = w.Pull(&git.PullOptions{
				RemoteName: "origin",
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return nil, err
			}
		} else if fetch {
			log.Printf("%s: Fetch %s", name, path)
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

func ParseRepositoryURL(u *url.URL) (urlPath, name string) {
	name = path.Base(u.Path)
	urlPath = strings.TrimSuffix(u.Path[1:], path.Ext(name))
	name = strings.TrimSuffix(name, path.Ext(name))
	return
}

func (repos *Repositories) CloneDependency(lib *Library, directory string, useHead bool) (clonePath string, err error) {
	libPath, name := ParseRepositoryURL(lib.URL)
	cached := path.Join(repos.Cache, name)

	p := path.Join(directory, name)
	if repos.NestedLayout {
		p = path.Join(directory, libPath)
	}

	pullCache := useHead
	if lib.Version == "*" || !repos.HasCommit(cached, lib.Version) {
		log.Printf("%s: Version mismatch, pulling", name)
		pullCache = true
	}
	if !pullCache {
		log.Printf("%s: Cache looks good", name)
	}

	_, err = repos.UpdateRepository(lib.Name, lib.URL.String(), cached, pullCache, false)
	if err != nil {
		return "", err
	}

	r, err := repos.UpdateRepository(lib.Name, cached, p, useHead, true)
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

	if useHead && lib.Version != "*" {
		if lib.Version != head {
			log.Printf("%s: Version changed: %v", name, head)
			lib.Version = head
			lib.Modified = true
		}
	}

	if lib.Version != "" {
		log.Printf("%s: Checkout out %s (head = %s)", name, lib.Version, head)
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
