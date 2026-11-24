package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  Stack
	}{
		{"dockerfile wins", []string{"Dockerfile", "package.json"}, StackDockerfile},
		{"node", []string{"package.json"}, StackNode},
		{"python requirements", []string{"requirements.txt"}, StackPython},
		{"go", []string{"go.mod"}, StackGo},
		{"java", []string{"pom.xml"}, StackJava},
		{"fallback", []string{"README.md"}, StackBuildpacks},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, file := range tt.files {
				if err := os.WriteFile(filepath.Join(root, file), []byte("fixture"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := Detect(root)
			if err != nil {
				t.Fatal(err)
			}
			if got.Stack != tt.want {
				t.Fatalf("stack = %q, want %q", got.Stack, tt.want)
			}
		})
	}
}
