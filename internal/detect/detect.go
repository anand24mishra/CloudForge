package detect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Stack string

const (
	StackDockerfile Stack = "dockerfile"
	StackNode       Stack = "node"
	StackPython     Stack = "python"
	StackGo         Stack = "go"
	StackJava       Stack = "java"
	StackBuildpacks Stack = "buildpacks"
)

type Result struct {
	Stack          Stack    `json:"stack"`
	Build          string   `json:"build"`
	Start          string   `json:"start"`
	Port           int      `json:"port"`
	Dockerfile     string   `json:"dockerfile,omitempty"`
	DependencyFile string   `json:"dependencyFile,omitempty"`
	Evidence       []string `json:"evidence"`
	Confidence     string   `json:"confidence"`
}

func (r Result) String() string {
	b, _ := json.MarshalIndent(r, "", "  ")
	return string(b)
}

func Detect(root string) (Result, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return Result{}, fmt.Errorf("read repository: %w", err)
	}
	files := make(map[string]bool, len(entries))
	for _, entry := range entries {
		files[entry.Name()] = true
	}
	if files["Dockerfile"] {
		return Result{Stack: StackDockerfile, Build: "docker build", Start: "image-defined", Port: 8080, Dockerfile: "Dockerfile", Evidence: []string{"Dockerfile"}, Confidence: "high"}, nil
	}
	checks := []struct {
		stack Stack
		file  string
		build string
		start string
		port  int
	}{
		{StackNode, "package.json", "npm ci", "npm start", 3000},
		{StackPython, "pyproject.toml", "pip install .", "python -m app", 8000},
		{StackPython, "requirements.txt", "pip install -r requirements.txt", "python app.py", 8000},
		{StackGo, "go.mod", "go build ./...", "./app", 8080},
		{StackJava, "pom.xml", "./mvnw package", "java -jar target/app.jar", 8080},
		{StackJava, "build.gradle", "./gradlew build", "java -jar build/libs/app.jar", 8080},
	}
	for _, check := range checks {
		if files[check.file] {
			result := Result{Stack: check.stack, Build: check.build, Start: check.start, Port: check.port, Evidence: []string{check.file}, Confidence: "medium"}
			if check.stack == StackPython {
				result.DependencyFile = check.file
			}
			return result, nil
		}
	}
	// Buildpacks is deliberately a safe fallback. pack/buildpacks will produce a useful
	// error later if the repository has no supported build plan.
	known := make([]string, 0, len(files))
	for file := range files {
		known = append(known, file)
	}
	sort.Strings(known)
	return Result{Stack: StackBuildpacks, Build: "pack build", Start: "image-defined", Port: 8080, Evidence: known, Confidence: "fallback"}, nil
}

func IsSupported(stack Stack) bool {
	switch stack {
	case StackDockerfile, StackNode, StackPython, StackGo, StackJava, StackBuildpacks:
		return true
	default:
		return false
	}
}

func HasFile(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}
