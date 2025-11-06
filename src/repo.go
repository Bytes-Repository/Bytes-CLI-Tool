package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"net/http"
	"io"
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
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "=") && strings.Contains(line, "[") {
			// New section, e.g., = Public [
			parts := strings.SplitN(line, " [", 2)
			section := strings.TrimPrefix(parts[0], "=")
			repo[section] = make(map[string]map[string]string)
			currentSection = section
			currentCategory = ""
		} else if strings.HasPrefix(line, "=") {
			// New category, e.g., = GUI [
			parts := strings.SplitN(line, " [", 2)
			category := strings.TrimPrefix(parts[0], "=")
			if currentSection != "" {
				repo[currentSection][category] = make(map[string]string)
				currentCategory = category
			}
		} else if strings.Contains(line, "=>") {
			// Package, e.g., obsidian => url
			parts := strings.SplitN(line, "=>", 2)
			name := strings.TrimSpace(parts[0])
			url := strings.TrimSpace(parts[1])
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
			if currentCategory != "" {
				currentCategory = ""
			} else {
				currentSection = ""
			}
		}
	}

	return repo, scanner.Err()
}
