package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const defaultNamespace = "cloudforge"

// App is the normalized deployment configuration used by the CLI, API, and generators.
type App struct {
	Name            string
	Namespace       string
	Repository      string
	ImageRepository string
	ImageTag        string
	Domain          string
	Port            int
	ContainerPort   int
	Replicas        int
	BuildStrategy   string
}

var validName = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func DefaultApp(repo, name string) App {
	if name == "" {
		name = NameFromRepo(repo)
	}
	return App{
		Name:            NormalizeName(name),
		Namespace:       defaultNamespace,
		Repository:      repo,
		ImageRepository: "ghcr.io/OWNER/" + NormalizeName(name),
		ImageTag:        "latest",
		Port:            8080,
		ContainerPort:   8080,
		Replicas:        1,
		BuildStrategy:   "dockerfile",
	}
}

func NameFromRepo(repo string) string {
	clean := strings.TrimSuffix(filepath.Base(strings.TrimSuffix(repo, "/")), ".git")
	if clean == "." || clean == string(filepath.Separator) || clean == "" {
		return "app"
	}
	return NormalizeName(clean)
}

func NormalizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.NewReplacer("_", "-", ".", "-", "/", "-").Replace(name)
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, "-")
	if name == "" {
		return "app"
	}
	if len(name) > fiftyThree {
		name = strings.Trim(name[:fiftyThree], "-")
	}
	return name
}

const fiftyThree = 53

func ValidateApp(app App) error {
	if !validName.MatchString(app.Name) || len(app.Name) > 63 {
		return fmt.Errorf("invalid app name %q: use lowercase DNS-1123 characters", app.Name)
	}
	if app.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if app.Port < 1 || app.Port > 65535 || app.ContainerPort < 1 || app.ContainerPort > 65535 {
		return fmt.Errorf("ports must be between 1 and 65535")
	}
	if app.Replicas < 1 {
		return fmt.Errorf("replicas must be at least 1")
	}
	return nil
}

func RepoPath(repo string) (string, error) {
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "git@") || strings.HasPrefix(repo, "ssh://") {
		return "", fmt.Errorf("remote repository %q must be cloned by the caller", repo)
	}
	path, err := filepath.Abs(repo)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("repository path: %w", err)
	}
	return path, nil
}
