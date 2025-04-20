package config

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	unixpath "path"
	"path/filepath"
	"regexp"
	"sync"
	"vend/internal/user"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/goccy/go-yaml"
)

type (
	Config struct {
		Version  string   `yaml:"version"`
		Location string   `yaml:"-"`
		Sources  []Source `yaml:"sources"`
	}

	Source struct {
		Url           string `yaml:"url"`
		ReferenceName string `yaml:"reference_name"`
	}
)

const configFileName = "vend.yaml"

func New() Config {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	return Config{
		Version:  "1.0.0",
		Location: filepath.Join(wd, configFileName),
		Sources:  []Source{},
	}
}

func Load() (c *Config, err error) {
	c = &Config{Sources: []Source{}}
	wd, err := os.Getwd()
	if err != nil {
		return c, fmt.Errorf("failed to get working directory: %w", err)
	}

	// look for config file in working directory, if not found, look one directory up until found or at root directory
	for {
		configPath := filepath.Join(wd, configFileName)
		if _, err := os.Stat(configPath); err == nil {
			f, err := os.Open(configPath)
			if err != nil {
				return c, fmt.Errorf("failed to open config file: %w", err)
			}
			defer f.Close()
			if err := yaml.NewDecoder(f).Decode(c); err != nil {
				return c, fmt.Errorf("failed to decode config file: %w", err)
			}
			c.Location = configPath
			break
		}
		up := filepath.Dir(wd)
		if up == wd {
			return c, fmt.Errorf("config file not found")
		}
		wd = up
	}

	return c, nil
}

var sourceRE = regexp.MustCompile(`^(.+)@([^@]+)$`)

func (c *Config) Add(source string) error {
	match := sourceRE.FindStringSubmatch(source)
	if len(match) != 3 {
		return fmt.Errorf("invalid source format, expected '<url>@≤tag>'")
	}
	c.Sources = append(c.Sources, Source{
		Url:           match[1],
		ReferenceName: match[2],
	})
	return nil
}

// var gitmodulesRE = regexp.MustCompile(`\[\s*submodule\s+"([^"]+)"\s*\]\n((?:\s+\w+\s*=\s*[\w\/:.&;?-]+)+)\n`)

func (c *Config) FromGitSubmodules() error {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	sms, err := wt.Submodules()
	if err != nil {
		return fmt.Errorf("failed to get submodules: %w", err)
	}
	if len(sms) == 0 {
		fmt.Fprintf(os.Stdout, "no submodules found\n")
		return nil
	}
	for _, sm := range sms {
		repo, err := sm.Repository()
		if err != nil {
			return fmt.Errorf("failed to get submodule repository, %s: %w", sm.Config().Name, err)
		}
		smUrl := sm.Config().URL
		for _, source := range c.Sources {
			if source.Url == smUrl {
				continue
			}
		}
		headRef, err := repo.Head()
		if err != nil {
			return fmt.Errorf("failed to get submodule head: %w", err)
		}
		refName := headRef.Name()
		s := Source{
			Url:           smUrl,
			ReferenceName: refName.String(),
		}

		cmd := exec.Command("git", "submodule", "deinit", "-f", sm.Config().Path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to deinit submodule %s: %w", sm.Config().Name, err)
		}

		c.Sources = append(c.Sources, s)
	}

	return nil
}

func (c *Config) Save() error {
	if c.Location == "" {
		return fmt.Errorf("config location not set")
	}
	f, err := os.Create(c.Location)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()
	if err := yaml.NewEncoder(f).Encode(c); err != nil {
		return fmt.Errorf("failed to encode config file: %w", err)
	}
	return nil
}

func (c *Config) contains(name string) bool {
	for _, source := range c.Sources {
		if source.ShortName() == name {
			return true
		}
	}
	return false
}

func (c *Config) Sync() {
	dirEntries, err := os.ReadDir(user.Location())
	if err != nil {
		fmt.Printf("failed to read directory: %v", err)
		return
	}
	for _, entry := range dirEntries {
		if entry.IsDir() {
			if !c.contains(entry.Name()) {
				fmt.Printf("removing directory %s\n", entry.Name())
				_ = os.RemoveAll(entry.Name())
			}
		} else {
			fi, err := entry.Info()
			if err != nil {
				fmt.Printf("failed to get file info: %v", err)
				continue
			}
			// is symlink?
			if fi.Mode().Type() == os.ModeSymlink {
				if !c.contains(entry.Name()) {
					fmt.Printf("removing symlink %s\n", entry.Name())
					_ = os.Remove(entry.Name())
				}
			} else {
				fmt.Printf("removing unexpected file %s\n", entry.Name())
				_ = os.Remove(entry.Name())
			}
			continue
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(len(c.Sources))
	for _, source := range c.Sources {
		go source.get(&wg)
	}
	wg.Wait()
}

func (s Source) ShortName() string {
	u, err := url.Parse(s.Url)
	if err != nil {
		return unixpath.Base(s.Url)
	}
	return unixpath.Base(u.Path)
}

func (s Source) Name() string {
	u, err := url.Parse(s.Url)
	if err != nil {
		return filepath.Join(unixpath.Base(s.Url), s.ReferenceName)
	}
	return filepath.Join(u.Host, unixpath.Clean(u.Path), s.ReferenceName)
}

func (s Source) get(wg *sync.WaitGroup) {
	defer wg.Done()
	dir := filepath.Join(user.Location(), s.Name())
	referenceName := plumbing.ReferenceName(s.ReferenceName)
	if _, err := os.Stat(dir); err == nil {
		// update the repository
		repo, err := git.PlainOpen(dir)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to open repository: %v", err)
			return
		}
		w, err := repo.Worktree()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to get worktree: %v", err)
			return
		}
		w.Checkout(&git.CheckoutOptions{
			Branch: referenceName,
			Create: false,
		})
		err = w.Pull(&git.PullOptions{ReferenceName: referenceName, Force: true})
		if err != nil {
			return
		}
	} else {
		// clone the repository
		git.PlainClone(dir, false, &git.CloneOptions{URL: s.Url, Depth: 1, ReferenceName: referenceName})
	}

	// link repository to vendored directory
	vendoredDir := filepath.Join("vendored", s.ShortName())
	err := os.Symlink(dir, vendoredDir)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to link repository: %v", err)
		return
	}
}
