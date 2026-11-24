package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anandmishra1/cloudforge/internal/config"
	"github.com/anandmishra1/cloudforge/internal/detect"
)

func TestGenerateNode(t *testing.T) {
	root := t.TempDir()
	app := config.DefaultApp(root, "sample-node")
	app.ImageRepository = "ghcr.io/example/sample-node"
	files, err := Generate(Options{App: app, Detection: detect.Result{Stack: detect.StackNode, Port: 3000}, OutputDir: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("generated %d files, want 3", len(files))
	}
	dockerfile, err := os.ReadFile(filepath.Join(root, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerfile), "node:22") {
		t.Fatalf("Dockerfile did not contain Node base image: %s", dockerfile)
	}
	workflow, err := os.ReadFile(filepath.Join(root, ".github/workflows/cloudforge.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(workflow), "linux/arm64") {
		t.Fatalf("workflow did not contain ARM build: %s", workflow)
	}
}

func TestGenerateDockerfileAppDoesNotOverwriteDockerfile(t *testing.T) {
	root := t.TempDir()
	app := config.DefaultApp(root, "sample")
	files, err := Generate(Options{App: app, Detection: detect.Result{Stack: detect.StackDockerfile}, OutputDir: root})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) == "Dockerfile" {
			t.Fatal("Dockerfile stack should not generate a Dockerfile")
		}
	}
}
