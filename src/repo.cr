package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Repo structure: map[section]map[category]map[name]url
type Repo map[string]map[string]map[string]string

func refreshRepo(url, localPath string) error {
	resp, err := http.Get(url)
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	repo := make(Repo)
	for section, secVal := range raw {
		secMap, ok := secVal.(map[string]interface{})
		if !ok {
			continue
		}
		repo[section] = make(map[string]map[string]string)
		for category, catVal := range secMap {
			if catVal == nil {
				repo[section][category] = make(map[string]string)
				continue
			}
			catMap, ok := catVal.(map[string]interface{})
			if !ok {
				continue
			}
			repo[section][category] = make(map[string]string)
			for name, urlVal := range catMap {
				url := ""
				if urlVal != nil {
					if s, ok := urlVal.(string); ok {
						url = s
					}
				}
				repo[section][category][name] = url
			}
		}
	}
	return repo, nil
}
