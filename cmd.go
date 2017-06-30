package main

import (
	"flag"
	"github.com/op/go-logging"
	"github.com/pdbogen/gosible/core"
)

func main() {
	log := logging.MustGetLogger("gosible")
	rootPath := flag.String("root", "", "root path under which targets, credentials, and tasks can be found")

	flag.Parse()

	gosibleCore := core.Core{
		Root: *rootPath,
	}

	if err := gosibleCore.Load(); err != nil {
		log.Fatalf("loading gosible payloads: %s", err)
	}

	if err := gosibleCore.Run(); err != nil {
		log.Fatalf("running gosible payloads: %s", err)
	}

	log.Infof("All done! %d execs, %d changes", gosibleCore.Execs, gosibleCore.Changes)
}
