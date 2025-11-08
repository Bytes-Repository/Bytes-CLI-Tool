package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func searchPackages(repo Repo, query string) {
	found := false
	header := headerStyle.Render("Search Results for: " + query)
	fmt.Println(header)
	for section, categories := range repo {
		fmt.Println(boldStyle.Render(section + ":"))
		for category, packages := range categories {
			if category != "" {
				fmt.Println("  " + infoStyle.Render(category + ":"))
			}
			for name, url := range packages {
				urlDisplay := url
				if url == "" {
					urlDisplay = "(no release yet)"
				}
				if strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
					fmt.Printf("    %s => %s\n", successStyle.Render(name), urlDisplay)
					found = true
				}
			}
		}
	}
	if !found {
		fmt.Println(warnStyle.Render("No packages found matching: " + query))
	}
}

func moveFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		os.Remove(dst)
		return err
	}

	out.Sync() // Ensure written to disk

	err = os.Chmod(dst, 0644)
	if err != nil {
		return err
	}

	return os.Remove(src)
}

func installPackage(repo Repo, pkg string, libDir string) bool {
	var url string
	found := false
	for _, categories := range repo {
		for _, packages := range categories {
			for name, u := range packages {
				if strings.ToLower(name) == strings.ToLower(pkg) {
					url = u
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		fmt.Println(errorStyle.Render("Package not found: " + pkg))
		return false
	}
	if url == "" {
		fmt.Println(errorStyle.Render("No release found for package: " + pkg))
		return false
	}
	dest := filepath.Join(libDir, pkg)
	tmpDest := filepath.Join(os.TempDir(), pkg+"-"+fmt.Sprintf("%d", time.Now().Unix()))
	fmt.Println(infoStyle.Render("Downloading " + pkg + " from " + url))
	err := downloadWithProgress(url, tmpDest)
	if err != nil {
		fmt.Println(errorStyle.Render("Error downloading: " + err.Error()))
		return false
	}
	// Remove existing if exists
	if _, err := os.Stat(dest); err == nil {
		if err := os.Remove(dest); err != nil {
			fmt.Println(errorStyle.Render("Error removing old version: " + err.Error()))
			os.Remove(tmpDest)
			return false
		}
	}
	// Move to lib dir using copy to handle cross-device
	if err := moveFile(tmpDest, dest); err != nil {
		fmt.Println(errorStyle.Render("Error installing: " + err.Error()))
		os.Remove(tmpDest)
		return false
	}
	fmt.Println(successStyle.Render("Installed " + pkg + " to " + libDir))
	return true
}

func removePackage(pkg string, libDir string) {
	path := filepath.Join(libDir, pkg)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println(warnStyle.Render("Package not installed: " + pkg))
		return
	}
	if err := os.Remove(path); err != nil {
		fmt.Println(errorStyle.Render("Error removing: " + err.Error()))
		return
	}
	fmt.Println(successStyle.Render("Removed " + pkg))
}

func updatePackages(libDir, localRepo string) {
	files, err := os.ReadDir(libDir)
	if err != nil {
		fmt.Println(errorStyle.Render("Error reading lib dir: " + err.Error()))
		return
	}
	repo, err := parseRepo(localRepo)
	if err != nil {
		fmt.Println(errorStyle.Render("Error parsing repo: " + err.Error()))
		return
	}
	updated := 0
	for _, file := range files {
		if !file.IsDir() {
			pkg := file.Name()
			fmt.Println(infoStyle.Render("Checking update for " + pkg))
			if installPackage(repo, pkg, libDir) {
				updated++
			}
		}
	}
	if updated == 0 {
		fmt.Println(warnStyle.Render("No packages installed to update."))
	} else {
		fmt.Println(successStyle.Render(fmt.Sprintf("Updated %d packages.", updated)))
	}
}
