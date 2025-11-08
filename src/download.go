package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type progressModel struct {
	prog    progress.Model
	spinner spinner.Model
	total   int64
	read    int64
	err     error
	done    bool
	url     string
	dest    string
	reader  io.Reader
	file    *os.File
}

func newProgressModel(url, dest string, reader io.Reader, total int64, file *os.File) progressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF69B4"))
	p := progress.New(
		progress.WithScaledGradient("#FF7CCB", "#FDFF8C"),
			  progress.WithWidth(50),
	)
	return progressModel{
		prog:    p,
		spinner: s,
		total:   total,
		url:     url,
		dest:    dest,
		reader:  reader,
		file:    file,
	}
}

func (m progressModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startDownload())
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "ctrl+c" || msg.String() == "q" || msg.String() == "esc" {
				m.err = fmt.Errorf("download cancelled")
				m.file.Close()
				os.Remove(m.dest)
				return m, tea.Quit
			}
		case updateProgress:
			m.read = msg.read
			percent := float64(m.read) / float64(m.total)
			cmd := m.prog.SetPercent(percent)
			if percent >= 1.0 {
				m.done = true
				return m, tea.Quit
			}
			return m, cmd
		case progress.FrameMsg:
			newModel, cmd := m.prog.Update(msg)
			if newModel, ok := newModel.(progress.Model); ok {
				m.prog = newModel
			}
			return m, cmd
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
	}
	return m, nil
}

func (m progressModel) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error()) + "\n"
	}
	pad := strings.Repeat(" ", (m.prog.Width/2)-len("Downloading...")/2)
	status := fmt.Sprintf("%s Downloading %s...\n\n%s\n\n%.2f%% of %d bytes\nPress q to quit",
			      m.spinner.View(),
			      m.url,
		       m.prog.View(),
			      m.prog.Percent()*100,
			      m.total,
	)
	return infoStyle.Render(pad + status)
}

func (m *progressModel) startDownload() tea.Cmd {
	return func() tea.Msg {
		pr := &progressReader{
			reader: m.reader,
			total:  m.total,
			model:  m,
		}
		_, m.err = io.Copy(m.file, pr)
		m.file.Close()
		if m.err != nil {
			os.Remove(m.dest)
			return nil
		}
		return updateProgress{read: m.total}
	}
}

type updateProgress struct {
	read int64
}

type progressReader struct {
	reader io.Reader
	total  int64
	read   int64
	model  *progressModel
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.read += int64(n)
	pr.model.read = pr.read
	return
}

func downloadWithProgress(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	m := newProgressModel(url, dest, resp.Body, resp.ContentLength, f)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}
	return m.err
}
