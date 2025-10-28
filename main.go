package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	repoURL     = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/bytes.io"
	hackerApps  = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/scripts.hacker"
	viraApps    = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/apps.vira"
	version     = "0.1.0"
	installType string
	lang        string
	name        string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "bytes",
		Short:   "Bytes Repository CLI Tool",
		Version: version,
	}

	websiteCmd := &cobra.Command{
		Use:   "website",
		Short: "Show the Bytes website",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("https://bytes-repository.github.io/bytes.io/website/home.html")
		},
	}

	joinCmd := &cobra.Command{
		Use:   "join",
		Short: "Information on how to join",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Chcesz dołączyć i tworzyć biblioteki dla hacker lang lub vira lang? Zgłoś się na https://github.com/Bytes-Repository/bytes.io/issues")
		},
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install libraries or apps",
		Run: func(cmd *cobra.Command, args []string) {
			p := tea.NewProgram(initialInstallModel())
			if _, err := p.Run(); err != nil {
				fmt.Println("Error running program:", err)
				os.Exit(1)
			}
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove libraries or apps",
		Run: func(cmd *cobra.Command, args []string) {
			p := tea.NewProgram(initialRemoveModel())
			if _, err := p.Run(); err != nil {
				fmt.Println("Error running program:", err)
				os.Exit(1)
			}
		},
	}

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update libraries",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Updating libraries... (implementation pending)")
			// TODO: Implement update logic, e.g., git pull in installed dirs
		},
	}

	autoremoveCmd := &cobra.Command{
		Use:   "autoremove",
		Short: "Remove unnecessary items",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Removing unnecessary items... (implementation pending)")
			// TODO: Implement autoremove
		},
	}

	autocleanCmd := &cobra.Command{
		Use:   "autoclean",
		Short: "Clean up unnecessary items",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Cleaning up... (implementation pending)")
			// TODO: Implement autoclean
		},
	}

	rootCmd.AddCommand(websiteCmd, joinCmd, installCmd, removeCmd, updateCmd, autoremoveCmd, autocleanCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Bubble Tea Models for Interactive UI

type choice struct {
	title string
	desc  string
}

type model struct {
	choices  []choice
	cursor   int
	selected map[int]struct{}
	stage    string // "type", "lang", "name"
}

func initialInstallModel() model {
	return model{
		stage:    "type",
		choices:  []choice{{title: "lib", desc: "Install a library"}, {title: "app", desc: "Install an application"}},
		selected: make(map[int]struct{}),
	}
}

func initialRemoveModel() model {
	return model{
		stage:    "type",
		choices:  []choice{{title: "lib", desc: "Remove a library"}, {title: "app", desc: "Remove an application"}},
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			if m.stage == "type" {
				installType = m.choices[m.cursor].title
				m.stage = "lang"
				m.choices = []choice{{title: "hacker", desc: "Hacker Lang"}, {title: "vira", desc: "Vira Lang"}}
				m.cursor = 0
				return m, nil
			} else if m.stage == "lang" {
				lang = m.choices[m.cursor].title
				m.stage = "name"
				// Load available names
				var err error
				m.choices, err = loadAvailable(installType, lang)
				if err != nil {
					fmt.Println("Error loading available items:", err)
					return m, tea.Quit
				}
				m.cursor = 0
				return m, nil
			} else if m.stage == "name" {
				name = m.choices[m.cursor].title
				if strings.HasPrefix(os.Args[1], "install") {
					err := performInstall(installType, lang, name, m.choices[m.cursor].desc)
					if err != nil {
						fmt.Println("Install error:", err)
					} else {
						fmt.Printf("Installed %s for %s lang: %s\n", installType, lang, name)
					}
				} else if strings.HasPrefix(os.Args[1], "remove") {
					err := performRemove(installType, lang, name)
					if err != nil {
						fmt.Println("Remove error:", err)
					} else {
						fmt.Printf("Removed %s for %s lang: %s\n", installType, lang, name)
					}
				}
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var title string
	switch m.stage {
	case "type":
		title = "Select type:"
	case "lang":
		title = "Select language:"
	case "name":
		title = "Select item:"
	}

	s := lipgloss.NewStyle().Bold(true).Render(title) + "\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s (%s)\n", cursor, choice.title, choice.desc)
	}

	s += "\nPress q to quit.\n"
	return lipgloss.NewStyle().Margin(1, 2).Render(s)
}

// Helper Functions

func loadAvailable(typ, lng string) ([]choice, error) {
	var url string
	var parser func(string) ([]choice, error)

	switch typ {
	case "lib":
		url = repoURL
		parser = parseLibs
	case "app":
		if lng == "hacker" {
			url = hackerApps
			parser = parseHackerApps
		} else if lng == "vira" {
			url = viraApps
			parser = parseViraApps
		}
	}

	if url == "" {
		return nil, fmt.Errorf("invalid type or lang")
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parser(string(body))
}

func parseLibs(content string) ([]choice, error) {
	var choices []choice
	var currentLang string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "=") {
			currentLang = strings.TrimSpace(strings.TrimPrefix(line, "="))
			continue
		}
		if strings.HasPrefix(line, "[") || strings.HasPrefix(line, "]") {
			continue
		}
		if parts := strings.SplitN(line, "=>", 2); len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			url := strings.TrimSpace(parts[1])
			if (lang == "hacker" && currentLang == "Hacker Lang") || (lang == "vira" && currentLang == "Vira") {
				choices = append(choices, choice{title: name, desc: url})
			}
		}
	}
	return choices, nil
}

func parseHackerApps(content string) ([]choice, error) {
	var choices []choice
	// Assuming format: [ script => repository ]
	re := regexp.MustCompile(`\[(.*?)\s*=>\s*(.*?)\]`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			choices = append(choices, choice{title: strings.TrimSpace(match[1]), desc: strings.TrimSpace(match[2])})
		}
	}
	return choices, nil
}

func parseViraApps(content string) ([]choice, error) {
	var choices []choice
	// Assuming format: @ [ repo name => github repo ]
	re := regexp.MustCompile(`@\s*\[(.*?)\s*=>\s*(.*?)\]`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			choices = append(choices, choice{title: strings.TrimSpace(match[1]), desc: strings.TrimSpace(match[2])})
		}
	}
	return choices, nil
}

func performInstall(typ, lng, nm, repo string) error {
	var targetDir string
	home, _ := os.UserHomeDir()

	if typ == "lib" {
		if lng == "hacker" {
			targetDir = filepath.Join(home, ".hackeros", "hacker-lang", "libs", nm)
		} else if lng == "vira" {
			targetDir = filepath.Join(home, ".vira", "libs", nm)
		}
	} else if typ == "app" {
		// Assuming apps install to bin or similar; for now, install to ~/.bytes/apps/
		targetDir = filepath.Join(home, ".bytes", "apps", lng, nm)
	}

	if targetDir == "" {
		return fmt.Errorf("invalid install path")
	}

	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return err
	}

	cmd := exec.Command("git", "clone", repo, targetDir)
	return cmd.Run()
}

func performRemove(typ, lng, nm string) error {
	var targetDir string
	home, _ := os.UserHomeDir()

	if typ == "lib" {
		if lng == "hacker" {
			targetDir = filepath.Join(home, ".hackeros", "hacker-lang", "libs", nm)
		} else if lng == "vira" {
			targetDir = filepath.Join(home, ".vira", "libs", nm)
		}
	} else if typ == "app" {
		targetDir = filepath.Join(home, ".bytes", "apps", lng, nm)
	}

	if targetDir == "" {
		return fmt.Errorf("invalid remove path")
	}

	return os.RemoveAll(targetDir)
}
