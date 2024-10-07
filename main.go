package main

import (
	"encoding/json"
	"fmt"
	"github.com/rabobank/npsb/conf"
	"github.com/rabobank/npsb/server"
	"github.com/rabobank/npsb/util"
	"os"
	"time"
)

func main() {
	fmt.Printf("npsb starting, version:%s, commit:%s\n", conf.VERSION, conf.COMMIT)

	conf.EnvironmentComplete()

	util.InitCFClient()

	initialize()

	server.StartServer()
}

// initialize npsb, reading the catalog json file, initializing a cf client, and check for the uaa client.
func initialize() {
	catalogFile := fmt.Sprintf("%s/catalog.json", conf.CatalogDir)
	file, err := os.ReadFile(catalogFile)
	if err != nil {
		fmt.Printf("failed reading catalog file %s: %s\n", catalogFile, err)
		os.Exit(8)
	}
	err = json.Unmarshal(file, &conf.Catalog)
	if err != nil {
		fmt.Printf("failed unmarshalling catalog file %s, error: %s\n", catalogFile, err)
		os.Exit(8)
	}

	// start the routine that checks consistency between the service instance (labels) and the actual network policies:
	go func() {
		for {
			util.SyncLabels2Policies()
			time.Sleep(time.Duration(conf.SyncIntervalSecs) * time.Second)
		}
	}()
}
