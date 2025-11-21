package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

const (
	repoURL           = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/bytes.io"
	localRepoPath     = "/tmp/bytes.io"
	pluginRepoURL     = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/plugins-repo.hacker"
	localPluginRepo   = "/tmp/plugins-repo.hacker"
	libDirSuffix      = "/.hackeros/hacker-lang/libs/"
	pluginDirSuffix   = "/.hackeros/hacker-lang/plugins/"
	appName           = "Bytes.io CLI Tool"
	version           = "0.5"
)

var (
	boldStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700")).Align(lipgloss.Center).Width(60).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#FF69B4")).BorderBottom(true)
	footerStyle  = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#A9A9A9")).Align(lipgloss.Center)
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	usr, err := user.Current()
	if err != nil {
		fmt.Println(errorStyle.Render("Error getting user home: " + err.Error()))
		os.Exit(1)
	}
	libDir := filepath.Join(usr.HomeDir, libDirSuffix)
	if err := os.MkdirAll(libDir, 0755); err != nil {
		fmt.Println(errorStyle.Render("Error creating lib dir: " + err.Error()))
		os.Exit(1)
	}
	pluginDir := filepath.Join(usr.HomeDir, pluginDirSuffix)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		fmt.Println(errorStyle.Render("Error creating plugin dir: " + err.Error()))
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
		case "plugin":
			if len(os.Args) < 3 {
				fmt.Println(errorStyle.Render("Usage: plugin <subcommand> [args]"))
				printPluginUsage()
				os.Exit(1)
			}
			subcmd := os.Args[2]
			if _, err := os.Stat(localPluginRepo); os.IsNotExist(err) {
				if err := refreshRepo(pluginRepoURL, localPluginRepo); err != nil {
					fmt.Println(errorStyle.Render("Error refreshing plugin repo: " + err.Error()))
					os.Exit(1)
				}
			}
			repo, err := parseRepo(localPluginRepo)
			if err != nil {
				fmt.Println(errorStyle.Render("Error parsing plugin repo: " + err.Error()))
				os.Exit(1)
			}
			switch subcmd {
				case "search":
					if len(os.Args) < 4 {
						fmt.Println(errorStyle.Render("Usage: plugin search <query>"))
						os.Exit(1)
					}
					query := os.Args[3]
					searchPackages(repo, query)
				case "install":
					if len(os.Args) < 4 {
						fmt.Println(errorStyle.Render("Usage: plugin install <plugin>"))
						os.Exit(1)
					}
					pkg := os.Args[3]
					installPackage(repo, pkg, pluginDir)
				case "remove":
					if len(os.Args) < 4 {
						fmt.Println(errorStyle.Render("Usage: plugin remove <plugin>"))
						os.Exit(1)
					}
					pkg := os.Args[3]
					removePackage(pkg, pluginDir)
				case "update":
					updatePackages(pluginDir, localPluginRepo)
				case "refresh":
					if err := refreshRepo(pluginRepoURL, localPluginRepo); err != nil {
						fmt.Println(errorStyle.Render("Error refreshing: " + err.Error()))
					} else {
						fmt.Println(successStyle.Render("Plugin repo refreshed successfully."))
					}
				default:
					fmt.Println(errorStyle.Render("Unknown plugin subcommand: " + subcmd))
					printPluginUsage()
					os.Exit(1)
			}
				case "search":
					if len(os.Args) < 3 {
						fmt.Println(errorStyle.Render("Usage: search <query>"))
						os.Exit(1)
					}
					if _, err := os.Stat(localRepoPath); os.IsNotExist(err) {
						if err := refreshRepo(repoURL, localRepoPath); err != nil {
							fmt.Println(errorStyle.Render("Error refreshing repo: " + err.Error()))
							os.Exit(1)
						}
					}
					query := os.Args[2]
					repo, err := parseRepo(localRepoPath)
					if err != nil {
						fmt.Println(errorStyle.Render("Error parsing repo: " + err.Error()))
						os.Exit(1)
					}
					searchPackages(repo, query)
				case "install":
					if len(os.Args) < 3 {
						fmt.Println(errorStyle.Render("Usage: install <package>"))
						os.Exit(1)
					}
					if _, err := os.Stat(localRepoPath); os.IsNotExist(err) {
						if err := refreshRepo(repoURL, localRepoPath); err != nil {
							fmt.Println(errorStyle.Render("Error refreshing repo: " + err.Error()))
							os.Exit(1)
						}
					}
					pkg := os.Args[2]
					repo, err := parseRepo(localRepoPath)
					if err != nil {
						fmt.Println(errorStyle.Render("Error parsing repo: " + err.Error()))
						os.Exit(1)
					}
					installPackage(repo, pkg, libDir)
				case "remove":
					if len(os.Args) < 3 {
						fmt.Println(errorStyle.Render("Usage: remove <package>"))
						os.Exit(1)
					}
					pkg := os.Args[2]
					removePackage(pkg, libDir)
				case "update":
					if _, err := os.Stat(localRepoPath); os.IsNotExist(err) {
						if err := refreshRepo(repoURL, localRepoPath); err != nil {
							fmt.Println(errorStyle.Render("Error refreshing repo: " + err.Error()))
							os.Exit(1)
						}
					}
					updatePackages(libDir, localRepoPath)
				case "refresh":
					if err := refreshRepo(repoURL, localRepoPath); err != nil {
						fmt.Println(errorStyle.Render("Error refreshing: " + err.Error()))
					} else {
						fmt.Println(successStyle.Render("Repo refreshed successfully."))
					}
				case "info":
					printInfo()
				case "how-to-use":
					printHowToUse()
				case "how-to-add":
					printHowToAdd()
				default:
					printUsage()
					os.Exit(1)
	}
}

func printUsage() {
	header := headerStyle.Render(appName + " v" + version)
	commands := `
	Commands:
	search <query> - Search for packages
	install <package> - Install a package
	remove <package> - Remove a package
	update - Update all installed libraries
	refresh - Refresh the repository
	info - Show tool information
	how-to-use - Show how to use and add custom repos
	how-to-add - Show how to add your repository

	Plugin Commands:
	plugin search <query> - Search for plugins
	plugin install <plugin> - Install a plugin
	plugin remove <plugin> - Remove a plugin
	plugin update - Update all installed plugins
	plugin refresh - Refresh the plugin repository
	`
	footer := footerStyle.Render("Created by HackerOS Team")
	fmt.Println(lipgloss.JoinVertical(lipgloss.Left, header, infoStyle.Render(commands), footer))
}

func printPluginUsage() {
	commands := `
	Plugin Commands:
	search <query> - Search for plugins
	install <plugin> - Install a plugin
	remove <plugin> - Remove a plugin
	update - Update all installed plugins
	refresh - Refresh the plugin repository
	`
	fmt.Println(infoStyle.Render(commands))
}

func printInfo() {
	info := `
	Bytes.io CLI Tool for Hacker Lang (HackerOS)
	Version: ` + version + `
	Repository: https://github.com/Bytes-Repository/bytes.io
	Libs installed in: ~/.hackeros/hacker-lang/libs/
	Plugins installed in: ~/.hackeros/hacker-lang/plugins/
	`
	fmt.Println(infoStyle.Render(info))
}

func printHowToUse() {
	guide := `
	How to use and add your own repo to bytes.io:
	1. Fork the bytes.io repository on GitHub.
	2. Add your library to the repository/bytes.io file in the Community section.
	3. Format: Community: { CATEGORY: { your-lib: https://your-release-url } }
	4. Create a pull request to the main repo.
	5. Once merged, your lib will be available via this tool.
	`
	fmt.Println(infoStyle.Render(guide))
	fmt.Println(successStyle.Render("Happy hacking!"))
}

func printHowToAdd() {
	guide := `
	How to add your repository:
	Zgłoś swoje repozytorium w https://github.com/Bytes-Repository/bytes.io/issues lub https://github.com/Bytes-Repository/bytes.io/discussions
	Alternatively, follow the how-to-use guide to submit via PR.
	`
	fmt.Println(infoStyle.Render(guide))
}
