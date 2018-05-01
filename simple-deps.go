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
	Configuration string
	UrlOrPath     string
	Version       string
	Modified      bool
	URL           *url.URL
}

type Dependencies struct {
	Libraries []*Library
}

func NewEmptyDependencies() *Dependencies {
	return &Dependencies{
		Libraries: make([]*Library, 0),
	}
}

func NewDependencies(libraries []*Library) *Dependencies {
	return &Dependencies{
		Libraries: libraries,
	}
}

func (d *Dependencies) Write(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
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
			Configuration: path,
			UrlOrPath:     urlOrPath,
			Version:       version,
			URL:           url,
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

func cloneDependency(lib *Library, config *configuration) (err error) {
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
			return err
		}
	} else {
		r, err := git.PlainOpen(p)
		if err != nil {
			return err
		}

		w, err := r.Worktree()
		if err != nil {
			return err
		}

		if config.UseLatest {
			err = w.Pull(&git.PullOptions{
				RemoteName: "origin",
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return err
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
						return err
					}

					log.Printf("Checkout out %s", lib.Version)
					err = w.Checkout(&git.CheckoutOptions{
						Hash:  plumbing.NewHash(lib.Version),
						Force: true,
					})
					if err != nil {
						return err
					}
				} else {
					log.Printf("%s: Already on %s", name, lib.Version)
				}
			}
		}

		ref, err := r.Head()
		if err != nil {
			return err
		}

		newVersion := ref.Hash().String()
		if lib.Version != newVersion {
			log.Printf("%s: Version changed: %v", name, newVersion)
			lib.Version = newVersion
			lib.Modified = true
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

	deps := NewEmptyDependencies()

	for _, configuration := range configs {
		err := deps.Read(configuration)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}

	for _, lib := range deps.Libraries {
		if lib.URL != nil {
			if err := cloneDependency(lib, &config); err != nil {
				log.Fatal(err)
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

	byConfiguration := make(map[string][]*Library)

	for _, lib := range deps.Libraries {
		if byConfiguration[lib.Configuration] == nil {
			byConfiguration[lib.Configuration] = make([]*Library, 0)
		}
		byConfiguration[lib.Configuration] = append(byConfiguration[lib.Configuration], lib)
	}

	for configuration, libs := range byConfiguration {
		modified := false
		for _, lib := range libs {
			if lib.Modified {
				modified = true
				break
			}
		}

		if modified {
			log.Printf("%s: Writing", configuration)
			deps := NewDependencies(libs)
			deps.Write(configuration)
		}
	}
}
