package main

import (
	"bufio"
	"flag"
	"gopkg.in/src-d/go-git.v4"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
)

type Library struct {
	Name string
}

type Dependencies struct {
	Libraries []*Library
}

func readDependencies(path string) (*Dependencies, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var libs []*Library
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		libs = append(libs, &Library{
			Name: line,
		})
	}
	deps := &Dependencies{
		Libraries: libs,
	}
	return deps, scanner.Err()
}

type configuration struct {
	Configuration string
	Directory     string
}

func main() {
	config := configuration{}
	flag.StringVar(&config.Configuration, "config", "arduino-libraries", "libraries file")
	flag.StringVar(&config.Directory, "dir", "./gitdeps", "where to cache libraries")

	flag.Parse()

	deps, err := readDependencies(config.Configuration)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	for _, lib := range deps.Libraries {
		if url, err := url.ParseRequestURI(lib.Name); err == nil {
			name := path.Base(url.Path)
			name = strings.TrimSuffix(name, path.Ext(name))
			p := path.Join(config.Directory, name)

			log.Printf("Checking %s %s", url, p)

			if _, err := os.Stat(p); os.IsNotExist(err) {
				log.Printf("Cloning %s %s", url, p)

				_, err := git.PlainClone(p, false, &git.CloneOptions{
					URL:      lib.Name,
					Progress: os.Stdout,
				})
				if err != nil {
					log.Fatalf("Error: %v", err)
				}
			} else {
				log.Printf("Fetching %s %s", url, p)

				r, err := git.PlainOpen(p)
				if err != nil {
					log.Fatalf("Error: %v", err)
				}

				err = r.Fetch(&git.FetchOptions{
					RemoteName: "origin",
				})
				if err != nil && err != git.NoErrAlreadyUpToDate {
					log.Fatalf("Error: %v", err)
				}
			}
		} else {
			if s, err := os.Stat(lib.Name); err == nil && s.IsDir() {
				log.Printf("Using directory %v", lib.Name)
			}
		}
	}
}
