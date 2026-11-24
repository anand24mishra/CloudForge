package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Command struct {
	Name string
	Args []string
}

func (c Command) String() string {
	return strings.Join(append([]string{c.Name}, c.Args...), " ")
}

func Run(ctx context.Context, command Command, stdout, stderr io.Writer) error {
	if command.Name == "" {
		return fmt.Errorf("command name is required")
	}
	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", command.String(), err)
	}
	return nil
}

func Output(ctx context.Context, command Command) (string, error) {
	var stdout, stderr bytes.Buffer
	if err := Run(ctx, command, &stdout, &stderr); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}
