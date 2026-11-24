package generate

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/anandmishra1/cloudforge/internal/config"
	"github.com/anandmishra1/cloudforge/internal/detect"
)

//go:embed templates/*
var templateFS embed.FS

type Options struct {
	App         config.App
	Detection   detect.Result
	OutputDir   string
	WorkflowRef string
}

type templateData struct {
	App       config.App
	Detection detect.Result
}

func Generate(options Options) ([]string, error) {
	if err := config.ValidateApp(options.App); err != nil {
		return nil, err
	}
	if !detect.IsSupported(options.Detection.Stack) {
		return nil, fmt.Errorf("unsupported detected stack %q", options.Detection.Stack)
	}
	if options.OutputDir == "" {
		return nil, fmt.Errorf("output directory is required")
	}
	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}
	data := templateData{App: options.App, Detection: options.Detection}
	files := []string{"values.yaml", ".github/workflows/cloudforge.yml"}
	if options.Detection.Stack != detect.StackDockerfile {
		files = append(files, "Dockerfile")
	}
	written := make([]string, 0, len(files))
	for _, name := range files {
		content, err := templateFS.ReadFile(filepath.Join("templates", name+".tmpl"))
		if err != nil {
			return nil, fmt.Errorf("read template %s: %w", name, err)
		}
		parsed, err := template.New(name).Funcs(template.FuncMap{
			"quote": func(value string) string { return fmt.Sprintf("%q", value) },
		}).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}
		target := filepath.Join(options.OutputDir, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", target, err)
		}
		if err := parsed.Execute(file, data); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("render %s: %w", target, err)
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
		written = append(written, target)
	}
	return written, nil
}
