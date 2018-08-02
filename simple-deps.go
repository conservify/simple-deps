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
	UseLocal      bool
}

func main() {
	o := options{}

	flag.StringVar(&o.Configuration, "config", "", "libraries file")
	flag.StringVar(&o.Directory, "dir", "./gitdeps", "where to cache libraries")
	flag.BoolVar(&o.UseHead, "use-head", false, "pull and use head revision of git repositories")
	flag.BoolVar(&o.UseLocal, "use-local", false, "check for and use local versions")

	flag.Parse()

	configs := make([]string, 0)
	if s, err := os.Stat("arduino-libraries"); err == nil && !s.IsDir() {
		configs = append(configs, "arduino-libraries")
	}
	if o.Configuration != "" {
		configs = append(configs, o.Configuration)
	}
	configs = append(configs, flag.Args()...)

	deps := NewEmptyDependencies()

	for _, configuration := range configs {
		err := deps.Read(configuration)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}

	err := deps.Refresh(o.Directory, o.UseHead)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	deps.SaveModified()
}
