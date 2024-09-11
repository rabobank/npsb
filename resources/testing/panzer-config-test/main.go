package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/rabobank/npsb/resources/testing/panzer-config-test/model"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

var (
	createBuffer bytes.Buffer
	deleteBuffer bytes.Buffer
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: org-spaces-info <directory containing spaceConfig.yml files>")
		os.Exit(1)
	}
	directory := os.Args[1]
	fmt.Printf("# Directory: %s\n", directory)

	if err := filepath.Walk(directory, generateCreateAndBinds); err != nil {
		fmt.Println(err)
	}
	fmt.Println(createBuffer.String())
	_, _ = os.Stderr.WriteString(deleteBuffer.String())
}

func generateCreateAndBinds(fullPath string, info os.FileInfo, err error) error {
	if err == nil && info.Name() == "network-policies.yml" {
		var file *os.File
		if file, err = os.Open(fullPath); err != nil {
			fmt.Printf("%s could not be opened: %s\n", fullPath, err)
			return nil
		} else {
			defer func() { _ = file.Close() }()
			decoder := yaml.NewDecoder(bufio.NewReader(file))
			decoder.KnownFields(true)
			spaceConfig := model.SpacePolicies{}
			if err = decoder.Decode(&spaceConfig); err != nil {
				fmt.Printf("%s could not be parsed: %s\n", fullPath, err)
			} else {
				pieces := strings.Split(fullPath, "/")
				orgName := pieces[len(pieces)-3]
				spaceName := pieces[len(pieces)-2]
				createBuffer.WriteString(fmt.Sprintf("\ncf t -o %s -s %s\n", orgName, spaceName))
				deleteBuffer.WriteString(fmt.Sprintf("\ncf t -o %s -s %s\n", orgName, spaceName))
				for targetIndex, target := range spaceConfig.Targets {
					for fromIndex, from := range target.From {
						if from.Org == "" && from.Space == "" {
							createBuffer.WriteString(fmt.Sprintf(" cf cs network-policies default src%d-%d -c '{\"type\":\"source\",\"name\":\"%s-src%d-%d\",\"scope\":\"local\",\"description\":\"doe_iets_leuks\"}'\n", targetIndex, fromIndex, spaceName, targetIndex, fromIndex))
							deleteBuffer.WriteString(fmt.Sprintf(" cf ds -f src%d-%d\n", targetIndex, fromIndex))
							for _, fromApp := range from.Apps {
								createBuffer.WriteString(fmt.Sprintf("  cf push -f ../deploy-test-apps/cf-statics/manifest.yml -p ../deploy-test-apps/cf-statics \"%s\"\n", fromApp))
								//createBuffer.WriteString(fmt.Sprintf("  cf push -f ../deploy-test-apps/cf-statics/manifest.yml -p ../deploy-test-apps/cf-statics \"%s\" --no-start\n", fromApp))
								deleteBuffer.WriteString(fmt.Sprintf("  cf d -f -r \"%s\"\n", fromApp))
								createBuffer.WriteString(fmt.Sprintf("  cf map-route \"%s\" apps.internal --hostname \"%s\"\n", fromApp, fromApp))
								createBuffer.WriteString(fmt.Sprintf("  cf bs %s src%d-%d\n", fromApp, targetIndex, fromIndex))
							}
						}
					}
					for _, targetApp := range target.Apps {
						createBuffer.WriteString(fmt.Sprintf("  cf push -f ../deploy-test-apps/cf-statics/manifest.yml -p ../deploy-test-apps/cf-statics \"%s\"\n", targetApp))
						//createBuffer.WriteString(fmt.Sprintf("  cf push -f ../deploy-test-apps/cf-statics/manifest.yml -p ../deploy-test-apps/cf-statics \"%s\" --no-start\n", targetApp))
						deleteBuffer.WriteString(fmt.Sprintf("  cf d -f -r \"%s\"\n", targetApp))
						createBuffer.WriteString(fmt.Sprintf("  cf map-route \"%s\" apps.internal --hostname \"%s\"\n", targetApp, targetApp))
						for fromIndex, _ := range target.From {
							createBuffer.WriteString(fmt.Sprintf(" cf cs network-policies default dest%d-%d -c '{\"type\":\"destination\",\"source\":\"%s-src%d-%d\"}'\n", targetIndex, fromIndex, spaceName, targetIndex, fromIndex))
							deleteBuffer.WriteString(fmt.Sprintf(" cf ds -f dest%d-%d\n", targetIndex, fromIndex))
							if target.Port == 0 {
								createBuffer.WriteString(fmt.Sprintf("  cf bs %s dest%d-%d\n", targetApp, targetIndex, fromIndex))
							} else {
								createBuffer.WriteString(fmt.Sprintf("  cf bs %s dest%d-%d -c '{\"port\":%d}'\n", targetApp, targetIndex, fromIndex, target.Port))
							}
						}
					}
				}
			}
		}
	}
	return nil
}
