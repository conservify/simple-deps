package main

import (
	"log"
	"os"
	"path"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func GetRepositoryHash(p string) (h plumbing.Hash, err error) {
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

func CloneDependency(lib *Library, directory string, useHead bool) (clonePath string, err error) {
	name := path.Base(lib.URL.Path)
	name = strings.TrimSuffix(name, path.Ext(name))
	p := path.Join(directory, name)

	if _, err := os.Stat(p); os.IsNotExist(err) {
		log.Printf("Cloning %s %s", lib.URL, p)

		_, err := git.PlainClone(p, false, &git.CloneOptions{
			URL:      lib.UrlOrPath,
			Progress: os.Stdout,
		})
		if err != nil {
			return "", err
		}
	} else {
		r, err := git.PlainOpen(p)
		if err != nil {
			return "", err
		}

		w, err := r.Worktree()
		if err != nil {
			return "", err
		}

		if useHead {
			err = w.Pull(&git.PullOptions{
				RemoteName: "origin",
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return "", err
			}
		} else {
			if lib.Version != "" {
				existing, err := GetRepositoryHash(p)
				if existing.String() != lib.Version {
					log.Printf("Fetching %s %s", lib.URL, p)
					err = r.Fetch(&git.FetchOptions{
						RemoteName: "origin",
					})
					if err != nil && err != git.NoErrAlreadyUpToDate {
						return "", err
					}

					log.Printf("Checkout out %s", lib.Version)
					err = w.Checkout(&git.CheckoutOptions{
						Hash:  plumbing.NewHash(lib.Version),
						Force: true,
					})
					if err != nil {
						return "", err
					}
				} else {
					log.Printf("%s: Already on %s", name, lib.Version)
				}
			}
		}

		ref, err := r.Head()
		if err != nil {
			return "", err
		}

		newVersion := ref.Hash().String()
		if lib.Version != newVersion {
			log.Printf("%s: Version changed: %v", name, newVersion)
			lib.Version = newVersion
			lib.Modified = true
		}
	}
	return p, nil
}
