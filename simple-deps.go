package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

type Library struct {
	UrlOrPath string
	Version   string
	URL       *url.URL
}

type Dependencies struct {
	Libraries []*Library
}

func NewDependencies() *Dependencies {
	return &Dependencies{
		Libraries: make([]*Library, 0),
	}
}

func (d *Dependencies) Write(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil
	}

	defer f.Close()

	for _, lib := range d.Libraries {
		if lib.Version != "" {
			f.WriteString(fmt.Sprintf("%s %s\n", lib.UrlOrPath, lib.Version))
		} else {
			f.WriteString(fmt.Sprintf("%s\n", lib.UrlOrPath))
		}
	}

	return nil

}

func (d *Dependencies) Read(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, " ")
		urlOrPath := fields[0]
		version := ""
		if len(fields) > 1 {
			version = fields[1]
		}
		url, _ := url.ParseRequestURI(urlOrPath)
		d.Libraries = append(d.Libraries, &Library{
			UrlOrPath: urlOrPath,
			Version:   version,
			URL:       url,
		})
	}
	return scanner.Err()
}

type configuration struct {
	Configuration string
	Directory     string
	UseLatest     bool
}

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

func cloneDependency(lib *Library, config *configuration) (modified bool, err error) {
	name := path.Base(lib.URL.Path)
	name = strings.TrimSuffix(name, path.Ext(name))
	p := path.Join(config.Directory, name)

	if _, err := os.Stat(p); os.IsNotExist(err) {
		log.Printf("Cloning %s %s", lib.URL, p)

		_, err := git.PlainClone(p, false, &git.CloneOptions{
			URL:      lib.UrlOrPath,
			Progress: os.Stdout,
		})
		if err != nil {
			return false, err
		}
	} else {
		r, err := git.PlainOpen(p)
		if err != nil {
			return false, err
		}

		w, err := r.Worktree()
		if err != nil {
			return false, err
		}

		if config.UseLatest {
			err = w.Pull(&git.PullOptions{
				RemoteName: "origin",
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return false, err
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
						return false, err
					}

					log.Printf("Checkout out %s", lib.Version)
					err = w.Checkout(&git.CheckoutOptions{
						Hash:  plumbing.NewHash(lib.Version),
						Force: true,
					})
					if err != nil {
						return false, err
					}
				} else {
					log.Printf("%s already on %s", name, lib.Version)
				}
			}
		}

		ref, err := r.Head()
		if err != nil {
			return false, err
		}

		newVersion := ref.Hash().String()
		if lib.Version != newVersion {
			log.Printf("Version changed: %v", newVersion)
			lib.Version = newVersion
			modified = true
		}
	}
	return
}

func main() {
	config := configuration{}
	flag.StringVar(&config.Configuration, "config", "", "libraries file")
	flag.StringVar(&config.Directory, "dir", "./gitdeps", "where to cache libraries")
	flag.BoolVar(&config.UseLatest, "use-latest", false, "use the latest version of libraries")

	flag.Parse()

	configs := make([]string, 0)
	if s, err := os.Stat("arduino-libraries"); err == nil && !s.IsDir() {
		configs = append(configs, "arduino-libraries")
	}
	if config.Configuration != "" {
		configs = append(configs, config.Configuration)
	}
	configs = append(configs, flag.Args()...)

	deps := NewDependencies()

	for _, configuration := range configs {
		err := deps.Read(configuration)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}

	modified := false

	for _, lib := range deps.Libraries {
		if lib.URL != nil {
			cloneModified, err := cloneDependency(lib, &config)
			if err != nil {
				log.Fatal(err)
			}
			if cloneModified {
				modified = true
			}
		} else {
			if s, err := os.Stat(lib.UrlOrPath); err == nil && s.IsDir() {
				version, err := GetRepositoryHash(lib.UrlOrPath)
				if err == nil {
					log.Printf("Using directory %v (%v)", lib.UrlOrPath, version)
				} else {
					log.Printf("Using directory %v", lib.UrlOrPath)
				}
			}
		}
	}

	if modified {
		log.Printf("Wrote new configuration")
		deps.Write(config.Configuration)
	}
}
