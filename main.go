package main

import (
	"flag"
	"log"
	"os"
)

type options struct {
	Configuration string
	Directory     string
	UseHead       bool
	AllowLocal    bool
	Write         bool
	NestedLayout  bool
}

func main() {
	o := options{}

	flag.StringVar(&o.Configuration, "config", "", "libraries file")
	flag.StringVar(&o.Directory, "dir", "./gitdeps", "where to cache libraries")
	flag.BoolVar(&o.NestedLayout, "nested", false, "create nested directory layout")
	flag.BoolVar(&o.UseHead, "use-head", false, "pull and use head revision of git repositories")
	flag.BoolVar(&o.AllowLocal, "allow-local", true, "check for adjacent local copies, otherwise require cloning")
	flag.BoolVar(&o.Write, "write", false, "write the configuration file")

	flag.Parse()

	if os.Getenv("SIMPLE_USE_HEAD") != "" {
		o.UseHead = true
	}

	if os.Getenv("SIMPLE_DEPS_WRITE") != "" {
		o.Write = true
	}

	configs := make([]string, 0)
	if o.Configuration != "" {
		configs = append(configs, o.Configuration)
	}
	configs = append(configs, flag.Args()...)

	repositories, err := NewRepositories(o.NestedLayout)
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

	err = deps.Refresh(o.Directory, repositories, o.UseHead, o.AllowLocal)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	deps.SaveModified(o.Write)
}
