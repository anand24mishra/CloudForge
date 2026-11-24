package config

import "testing"

func TestNameFromRepo(t *testing.T) {
	tests := map[string]string{
		"https://github.com/example/my-app.git": "my-app",
		"./examples/node_app":                   "node-app",
		"repo":                                  "repo",
	}
	for input, want := range tests {
		if got := NameFromRepo(input); got != want {
			t.Errorf("NameFromRepo(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestValidateApp(t *testing.T) {
	valid := DefaultApp("example", "demo-app")
	if err := ValidateApp(valid); err != nil {
		t.Fatalf("valid app rejected: %v", err)
	}
	invalid := valid
	invalid.Name = "Demo_App"
	if err := ValidateApp(invalid); err == nil {
		t.Fatal("invalid app name was accepted")
	}
}
