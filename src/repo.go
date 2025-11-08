package main

import (
	"fmt"
	"os"
	"net/http"
	"io"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Repo structure: map[section]map[category]map[name]url
type Repo map[string]map[string]map[string]string

func refreshRepo(localPath string) error {
	resp, err := http.Get(repoURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func parseRepo(path string) (Repo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, err
	}
	repo := make(Repo)
	for section, secValue := range raw {
		sectionMap := make(map[string]map[string]string)
		if secMap, ok := secValue.(map[string]interface{}); ok {
			for catOrPkg, value := range secMap {
				if catMap, ok := value.(map[string]interface{}); ok {
					pkgMap := make(map[string]string)
					for pkg, url := range catMap {
						if url == nil {
							pkgMap[pkg] = ""
						} else {
							pkgMap[pkg] = url.(string)
						}
					}
					sectionMap[catOrPkg] = pkgMap
				} else {
					// Direct package under empty category
					if _, ok := sectionMap[""]; !ok {
						sectionMap[""] = make(map[string]string)
					}
					if value == nil {
						sectionMap[""][catOrPkg] = ""
					} else {
						sectionMap[""][catOrPkg] = value.(string)
					}
				}
			}
		}
		repo[section] = sectionMap
	}
	return repo, nil
}
