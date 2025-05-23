package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	unixpath "path"
	"path/filepath"
	"regexp"
	"strings"
	"vend/internal/sudo"
	"vend/internal/user"

	"github.com/go-git/go-git/v5"
	"github.com/goccy/go-yaml"
)

type (
	Config struct {
		Version  uint              `yaml:"version"`
		Scripts  map[string]string `yaml:"scripts"`
		Location string            `yaml:"-"`
		Sources  []Source          `yaml:"sources"`
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
		Version: 1,
		Scripts: map[string]string{
			"foo": `echo "foo"`,
		},
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
		return fmt.Errorf("invalid source format, expected '<url>@<tag>'")
	}
	for _, s := range c.Sources {
		if s.Url == match[1] {
			return fmt.Errorf("source %s exists already", source)
		}
	}
	c.Sources = append(c.Sources, Source{
		Url:           match[1],
		ReferenceName: match[2],
	})
	return nil
}

func (c *Config) Remove(source string) error {
	match := sourceRE.FindStringSubmatch(source)
	if len(match) != 3 {
		for i, s := range c.Sources {
			if s.Url == source || s.Name() == source || s.ShortName() == source {
				c.Sources = append(c.Sources[:i], c.Sources[i+1:]...)
				return nil
			}
		}
	}
	for i, s := range c.Sources {
		if s.Url == match[1] {
			c.Sources = append(c.Sources[:i], c.Sources[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("source %s not found", source)
}

func (c *Config) Run(script string, args []string) error {
	if s, ok := c.Scripts[script]; ok {
		if err := run(s, args); err != nil {
			return errors.Join(errors.New("failed to run script"), err)
		}
		return nil
	}
	return fmt.Errorf("script %s not found", script)
}

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

func (c *Config) Sync() {
	vendoredDir := "vendored"
	_ = os.MkdirAll(vendoredDir, 0755)
	dirEntries, err := os.ReadDir(vendoredDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read directory: %v", err)
		return
	}
	for _, entry := range dirEntries {
		entryName := filepath.Join(vendoredDir, entry.Name())
		fi, err := entry.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get file info: %v", err)
			continue
		}

		// is symlink?
		if fi.Mode().Type() == os.ModeSymlink {
			if err := os.Remove(entryName); err != nil {
				fmt.Fprintf(os.Stderr, "failed to remove symlink %s: %v\n", entryName, err)
			}
		} else if entry.IsDir() {
			_ = os.RemoveAll(entryName)
		} else {
			if err := os.Remove(entryName); err != nil {
				fmt.Fprintf(os.Stderr, "failed to remove %s: %v\n", entryName, err)
			}
		}
	}

	CloneMultiple(c.Sources)

	wd, _ := os.Getwd()
	linkData := make([]sudo.LinkData, 0, len(c.Sources))
	for _, source := range c.Sources {
		linkData = append(linkData, sudo.LinkData{
			Old: source.DestPath(),
			New: filepath.Join(wd, "vendored", source.ShortName()),
		})
	}
	sudo.Link(linkData)
}

func (s Source) ShortName() string {
	u, err := url.Parse(s.Url)
	if err != nil {
		return strings.TrimSuffix(unixpath.Base(s.Url), ".git")
	}
	return strings.TrimSuffix(unixpath.Base(u.Path), ".git")
}

func (s Source) Name() string {
	u, err := url.Parse(s.Url)
	if err != nil {
		return filepath.Join(strings.TrimSuffix(unixpath.Base(s.Url), ".git"), s.ReferenceName)
	}
	return filepath.Join(u.Host, strings.TrimSuffix(u.Path, ".git"), s.ReferenceName)
}

func (s Source) DestPath() string {
	return filepath.Join(user.Location(), s.Name())
}
