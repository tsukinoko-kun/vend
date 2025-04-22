package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type (
	repoProgressWriter struct {
		index      int
		progressCh chan<- any
	}

	progressMsg struct {
		Index   int
		Percent float64
	}

	submoduleMsg struct {
		Index   int
		Message string
	}

	doneMsg struct {
		Index int
		Error error
	}

	repoProgress struct {
		source                Source
		progress              progress.Model
		percent               float64
		done                  bool
		err                   error
		downloadingSubmodules bool
		statusMessage         string
	}

	model struct {
		repos    []repoProgress
		quitting bool
	}
)

func (pw *repoProgressWriter) Write(p []byte) (int, error) {
	msg := string(p)

	// Check for submodule operations - this needs to be more comprehensive
	if strings.Contains(msg, "Submodule") ||
		strings.Contains(msg, "submodule") ||
		strings.Contains(msg, "git submodule") {
		pw.progressCh <- submoduleMsg{
			Index:   pw.index,
			Message: strings.TrimSpace(msg),
		}
		return len(p), nil
	}

	// Parse percentage from strings like "Receiving objects:  15% (2734/18236)"
	if percent, ok := extractPercentage(msg); ok {
		pw.progressCh <- progressMsg{
			Index:   pw.index,
			Percent: percent,
		}
	}

	return len(p), nil
}

func extractPercentage(msg string) (float64, bool) {
	re := regexp.MustCompile(`(\d+)%`)
	matches := re.FindStringSubmatch(msg)
	if len(matches) == 2 {
		if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return percent / 100.0, true
		}
	}
	return 0, false
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		for i := range m.repos {
			m.repos[i].progress.Width = min(msg.Width-4, 80)
		}

	case progressMsg:
		if msg.Index >= 0 && msg.Index < len(m.repos) && !m.repos[msg.Index].done {
			m.repos[msg.Index].percent = msg.Percent
			progressModel, cmd := m.repos[msg.Index].progress.Update(progress.FrameMsg{})
			m.repos[msg.Index].progress = progressModel.(progress.Model)
			return m, cmd
		}

	case submoduleMsg:
		if msg.Index >= 0 && msg.Index < len(m.repos) && !m.repos[msg.Index].done {
			m.repos[msg.Index].downloadingSubmodules = true
			m.repos[msg.Index].statusMessage = msg.Message
			return m, nil
		}

	case doneMsg:
		if msg.Index >= 0 && msg.Index < len(m.repos) {
			m.repos[msg.Index].done = true
			m.repos[msg.Index].err = msg.Error
			m.repos[msg.Index].percent = 1.0
			m.repos[msg.Index].downloadingSubmodules = false
			m.repos[msg.Index].statusMessage = ""

			// Check if all repositories are done
			allDone := true
			for _, repo := range m.repos {
				if !repo.done {
					allDone = false
					break
				}
			}

			if allDone {
				return m, tea.Quit
			}
		}

	case progress.FrameMsg:
		cmds := make([]tea.Cmd, len(m.repos))
		for i := range m.repos {
			progressModel, cmd := m.repos[i].progress.Update(msg)
			m.repos[i].progress = progressModel.(progress.Model)
			cmds[i] = cmd
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Quitting...\n"
	}

	s := ""
	for i, repo := range m.repos {
		var status string

		if repo.done {
			if repo.err != nil {
				status = fmt.Sprintf(" [ERROR: %v]", repo.err)
				repo.progress.FullColor = "9"
			} else {
				status = " [DONE]"
				repo.progress.FullColor = "10"
			}
		} else if repo.downloadingSubmodules {
			// This status is shown even when the progress bar is at 100%
			status = " [SUBMODULES]"
			repo.progress.FullColor = "12"
		} else if repo.percent >= 0.99 {
			// For repositories at 99%+ but not marked as submodule downloading or done
			status = " [FINISHING]"
			repo.progress.FullColor = "12"
		}

		s += fmt.Sprintf("%d. %s%s\n", i+1, repo.source.Url, status)
		s += "   " + repo.progress.ViewAs(repo.percent) + "\n"

		// Show submodule status message if available
		if repo.downloadingSubmodules && repo.statusMessage != "" {
			s += "   " + repo.statusMessage + "\n"
		}

		s += "\n"
	}

	s += "Press q to quit.\n"
	return s
}

// This function is critical for understanding the full clone flow
func cloneRepository(source Source, index int, progressCh chan<- any, doneCh chan<- doneMsg) {
	dest := source.DestPath()

	// Check if repository already exists
	if _, err := os.Stat(dest); err == nil {
		doneCh <- doneMsg{Index: index, Error: nil}
		return
	}

	// Create progress writer
	progress := &repoProgressWriter{
		index:      index,
		progressCh: progressCh,
	}

	// Clone the repository with submodules
	referenceName := plumbing.ReferenceName(source.ReferenceName)

	// Send initial status message
	progressCh <- progressMsg{Index: index, Percent: 0.0}

	// We specifically enable recursive submodule cloning, which will trigger submodule operations
	_, err := git.PlainClone(dest, false, &git.CloneOptions{
		URL:               source.Url,
		Depth:             1,
		ReferenceName:     referenceName,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          progress,
	})

	// Only mark as done after all operations, including submodules, are complete
	doneCh <- doneMsg{Index: index, Error: err}
}

func CloneMultiple(sources []Source) error {
	if len(sources) == 0 {
		return nil
	}

	// Create channels for progress updates and completion
	progressCh := make(chan any)
	doneCh := make(chan doneMsg)

	// Create progress bars for each repository
	repos := make([]repoProgress, len(sources))
	for i, source := range sources {
		repos[i] = repoProgress{
			source:   source,
			progress: progress.New(progress.WithSolidFill("11")),
			percent:  0,
			done:     false,
		}
	}

	// Create the model
	m := model{
		repos:    repos,
		quitting: false,
	}

	// Start the bubbletea program
	p := tea.NewProgram(m)

	// Start the goroutines to clone repositories in parallel
	var wg sync.WaitGroup
	for i, source := range sources {
		wg.Add(1)
		go func(src Source, idx int) {
			defer wg.Done()
			cloneRepository(src, idx, progressCh, doneCh)
		}(source, i)
	}

	// Close the done channel when all goroutines complete
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	// Forward messages to the program
	go func() {
		for {
			select {
			case msg, ok := <-progressCh:
				if !ok {
					return
				}
				p.Send(msg)
			case msg, ok := <-doneCh:
				if !ok {
					return
				}
				p.Send(msg)
			}
		}
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		return err
	}

	// Check for errors
	for _, repo := range repos {
		if repo.err != nil {
			return repo.err
		}
	}

	return nil
}
