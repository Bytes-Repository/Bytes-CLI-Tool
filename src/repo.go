package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Repo structure: map[section]map[category]map[name]url
type Repo map[string]map[string]map[string]string

const (
	repoURL = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/bytes.io"
)

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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	repo := make(Repo)
	var currentSection string
	var currentCategory string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fullLine := scanner.Text()
		line := strings.TrimSpace(fullLine)
		if line == "" {
			continue
		}
		indentLevel := len(fullLine) - len(line)
		if strings.HasPrefix(line, "=") && strings.Contains(line, "[") {
			parts := strings.SplitN(line, " [", 2)
			name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(parts[0]), "="))
			if indentLevel == 0 {
				// New section
				repo[name] = make(map[string]map[string]string)
				currentSection = name
				currentCategory = ""
			} else {
				// New category
				if currentSection != "" {
					repo[currentSection][name] = make(map[string]string)
					currentCategory = name
				}
			}
		} else if strings.Contains(line, "=>") {
			// Package
			parts := strings.SplitN(line, "=>", 2)
			name := strings.TrimSpace(parts[0])
			url := ""
			if len(parts) > 1 {
				url = strings.TrimSpace(parts[1])
			}
			if name == "" {
				continue
			}
			if currentSection != "" && currentCategory != "" {
				repo[currentSection][currentCategory][name] = url
			} else if currentSection != "" {
				// Direct package in section
				if _, ok := repo[currentSection][""]; !ok {
					repo[currentSection][""] = make(map[string]string)
				}
				repo[currentSection][""][name] = url
			}
		} else if line == "]" {
			// End of category or section
			if indentLevel > 0 {
				currentCategory = ""
			} else {
				currentSection = ""
			}
		}
	}
	return repo, scanner.Err()
}
