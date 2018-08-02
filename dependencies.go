package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

type Library struct {
	Configuration string
	UrlOrPath     string
	Version       string
	RelativePath  string
	Recurse       bool
	Name          string
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
		recurseFlag := "norecurse"
		if lib.Recurse {
			recurseFlag = "recurse"
		}
		f.WriteString(fmt.Sprintf("%s %s %s %s\n", lib.UrlOrPath, lib.Version, lib.RelativePath, recurseFlag))
	}

	return nil
}

func (d *Dependencies) Read(fn string) error {
	file, err := os.Open(fn)
	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, " ")
		urlOrPath := fields[0]
		version := "*"
		relativePath := "/"
		recurse := false
		if len(fields) > 1 {
			version = fields[1]
		}
		if len(fields) > 2 {
			relativePath = fields[2]
		}
		if len(fields) > 3 {
			recurse = fields[3] == "recurse"
		}
		url, _ := url.ParseRequestURI(urlOrPath)
		name := ""
		if url != nil {
			name = path.Base(url.Path)
			name = strings.TrimSuffix(name, path.Ext(name))
		} else {
			name = path.Base(urlOrPath)
		}
		d.Libraries = append(d.Libraries, &Library{
			Configuration: fn,
			UrlOrPath:     urlOrPath,
			Version:       version,
			Name:          name,
			RelativePath:  relativePath,
			Recurse:       recurse,
			URL:           url,
		})
	}
	return scanner.Err()
}

func (d *Dependencies) SaveModified(force bool) error {
	byConfiguration := make(map[string][]*Library)

	for _, lib := range d.Libraries {
		if byConfiguration[lib.Configuration] == nil {
			byConfiguration[lib.Configuration] = make([]*Library, 0)
		}
		byConfiguration[lib.Configuration] = append(byConfiguration[lib.Configuration], lib)
	}

	for configuration, libs := range byConfiguration {
		modified := force
		for _, lib := range libs {
			if lib.Modified {
				modified = true
				break
			}
		}

		if modified {
			log.Printf("Writing %s", configuration)
			deps := NewDependencies(libs)
			if err := deps.Write(configuration); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkForLocalOverride(lib *Library) (string, error) {
	expected := path.Join("../", lib.Name)
	if s, err := os.Stat(expected); err == nil && !s.Mode().IsRegular() {
		abs, err := filepath.Abs(expected)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	return "", nil
}

type DependencyInfo struct {
	Name      string
	Path      string
	Recursive bool
}

type TemplateData struct {
	Dependencies []*DependencyInfo
}

func (d *Dependencies) Refresh(directory string, repos *Repositories, useHead bool) error {
	templateDatas := make([]*DependencyInfo, 0)
	project := "./"

	for _, lib := range d.Libraries {
		dependencyPath, err := checkForLocalOverride(lib)
		if err != nil {
			return err
		}
		if dependencyPath == "" {
			if lib.URL != nil {
				clonePath, err := repos.CloneDependency(lib, directory, useHead)
				if err != nil {
					return err
				}
				dependencyPath = clonePath
			} else {
				if s, err := os.Stat(lib.UrlOrPath); err == nil && s.IsDir() {
					version, err := repos.GetRepositoryHash(lib.UrlOrPath)
					if err == nil {
						log.Printf("Using directory %v (%v)", lib.UrlOrPath, version)
					} else {
						log.Printf("Using directory %v", lib.UrlOrPath)
					}
				}
			}
		}

		dependencyPath, err = filepath.Abs(dependencyPath)
		if err != nil {
			return err
		}

		log.Printf("Dependency: %s = %s", lib.UrlOrPath, dependencyPath)

		templateDatas = append(templateDatas, &DependencyInfo{
			Name: lib.Name,
			Path: dependencyPath,
		})

		project = filepath.Dir(lib.Configuration)
	}

	executable, err := os.Executable()
	if err != nil {
		panic(err)
	}
	dir := filepath.Dir(executable)

	templateData, err := ioutil.ReadFile(filepath.Join(dir, "dependencies.cmake.template"))
	if err != nil {
		return err
	}

	template, err := template.New("dependencies.cmake").Parse(string(templateData))
	if err != nil {
		return err
	}

	dependenciesPath := filepath.Join(project, "dependencies.cmake")
	log.Printf("Writing %s", dependenciesPath)

	dependenciesFile, err := os.Create(dependenciesPath)
	if err != nil {
		return err
	}

	defer dependenciesFile.Close()

	data := TemplateData{
		Dependencies: templateDatas,
	}

	err = template.Execute(dependenciesFile, data)
	if err != nil {
		return err
	}

	return nil
}
