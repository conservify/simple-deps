package main

import (
	"flag"
	"log"
)

type options struct {
	Configuration string
	Directory     string
	UseHead       bool
	Write         bool
}

func main() {
	o := options{}

	flag.StringVar(&o.Configuration, "config", "", "libraries file")
	flag.StringVar(&o.Directory, "dir", "./gitdeps", "where to cache libraries")
	flag.BoolVar(&o.UseHead, "use-head", false, "pull and use head revision of git repositories")
	flag.BoolVar(&o.Write, "write", false, "write the configuration file")

	flag.Parse()

	configs := make([]string, 0)
	if o.Configuration != "" {
		configs = append(configs, o.Configuration)
	}
	configs = append(configs, flag.Args()...)

	repositories, err := NewRepositories()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	deps := NewEmptyDependencies()

	for _, configuration := range configs {
		err := deps.Read(configuration)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}

	err = deps.Refresh(o.Directory, repositories, o.UseHead)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	deps.SaveModified(o.Write)
}
