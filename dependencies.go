package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
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

func (d *Dependencies) SaveModified() error {
	byConfiguration := make(map[string][]*Library)

	for _, lib := range d.Libraries {
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
			if err := deps.Write(configuration); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *Dependencies) Refresh(directory string, useHead bool) error {
	for _, lib := range d.Libraries {
		if lib.URL != nil {
			if err := CloneDependency(lib, directory, useHead); err != nil {
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

	return nil
}
