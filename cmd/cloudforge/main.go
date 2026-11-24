package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/anandmishra1/cloudforge/internal/config"
	"github.com/anandmishra1/cloudforge/internal/detect"
	"github.com/anandmishra1/cloudforge/internal/generate"
	"github.com/anandmishra1/cloudforge/internal/k8s"
	"github.com/anandmishra1/cloudforge/internal/runner"
	"github.com/spf13/cobra"
)

type cliOptions struct {
	namespace   string
	context     string
	kubeconfig  string
	kindCluster string
}

func main() {
	options := &cliOptions{}
	root := &cobra.Command{Use: "cloudforge", Short: "A local-first Git-to-Kubernetes PaaS", SilenceUsage: true}
	root.PersistentFlags().StringVarP(&options.namespace, "namespace", "n", "cloudforge", "Kubernetes namespace")
	root.PersistentFlags().StringVar(&options.context, "context", "", "Kubernetes context")
	root.PersistentFlags().StringVar(&options.kubeconfig, "kubeconfig", "", "kubeconfig path")
	root.PersistentFlags().StringVar(&options.kindCluster, "kind-cluster", "", "kind cluster name for loading locally built images")
	root.AddCommand(newDetectCommand())
	root.AddCommand(newInitCommand())
	root.AddCommand(newDeployCommand(options))
	root.AddCommand(newStatusCommand(options))
	root.AddCommand(newLogsCommand(options))
	root.AddCommand(newRollbackCommand(options))
	root.AddCommand(newDestroyCommand(options))
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "cloudforge:", err)
		os.Exit(1)
	}
}

func newDetectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "detect <repo>",
		Short: "Detect an application stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cleanup, err := localRepo(args[0])
			if err != nil {
				return err
			}
			defer cleanup()
			result, err := detect.Detect(root)
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), result)
		},
	}
}

func newInitCommand() *cobra.Command {
	var name, output string
	command := &cobra.Command{Use: "init <repo>", Short: "Generate deployment files into a local working copy", Args: cobra.ExactArgs(1)}
	command.Flags().StringVar(&name, "name", "", "application name")
	command.Flags().StringVar(&output, "output-dir", "", "directory to write generated files (defaults to the repository)")
	command.RunE = func(cmd *cobra.Command, args []string) error {
		root, cleanup, err := localRepo(args[0])
		if err != nil {
			return err
		}
		defer cleanup()
		if output == "" && isRemote(args[0]) {
			return errors.New("remote init requires --output-dir; CloudForge never pushes generated files")
		}
		result, err := detect.Detect(root)
		if err != nil {
			return err
		}
		app := config.DefaultApp(root, name)
		if output == "" {
			output = root
		}
		files, err := generate.Generate(generate.Options{App: app, Detection: result, OutputDir: output})
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "detected %s; generated:\n", result.Stack)
		for _, file := range files {
			fmt.Fprintln(cmd.OutOrStdout(), "-", file)
		}
		return nil
	}
	return command
}

func newDeployCommand(options *cliOptions) *cobra.Command {
	var name, tag string
	var build, useBuildpacks bool
	command := &cobra.Command{Use: "deploy <repo>", Short: "Build and deploy a local repository with Helm", Args: cobra.ExactArgs(1)}
	command.Flags().StringVar(&name, "name", "", "application name")
	command.Flags().StringVar(&tag, "tag", "local", "image tag")
	command.Flags().BoolVar(&build, "build", true, "build the image before deploying")
	command.Flags().BoolVar(&useBuildpacks, "buildpacks", false, "force Cloud Native Buildpacks instead of generating/using a Dockerfile")
	command.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Minute)
		defer cancel()
		root, cleanup, err := localRepo(args[0])
		if err != nil {
			return err
		}
		defer cleanup()
		result, err := detect.Detect(root)
		if err != nil {
			return err
		}
		app := config.DefaultApp(root, name)
		app.Namespace, app.ImageTag = options.namespace, tag
		app.ImageRepository = "cloudforge/" + app.Name
		if useBuildpacks {
			result.Stack = detect.StackBuildpacks
		}
		if !useBuildpacks && !detect.HasFile(root, "Dockerfile") && result.Stack != detect.StackBuildpacks {
			if _, err := generate.Generate(generate.Options{App: app, Detection: result, OutputDir: root}); err != nil {
				return err
			}
		}
		if build {
			if err := buildImage(ctx, root, app.ImageRepository+":"+tag, result); err != nil {
				return err
			}
			kindCluster := options.kindCluster
			if kindCluster == "" && strings.HasPrefix(options.context, "kind-") {
				kindCluster = strings.TrimPrefix(options.context, "kind-")
			}
			if kindCluster != "" {
				if err := runner.Run(ctx, runner.Command{Name: "kind", Args: []string{"load", "docker-image", app.ImageRepository + ":" + tag, "--name", kindCluster}}, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
					return err
				}
			}
		}
		helmArgs := []string{"upgrade", "--install", app.Name, "charts/cloudforge-app", "--namespace", app.Namespace, "--create-namespace", "--set", "nameOverride=" + app.Name, "--set", "fullnameOverride=" + app.Name, "--set", "image.repository=" + app.ImageRepository, "--set", "image.tag=" + tag, "--set", "image.pullPolicy=IfNotPresent", "--set", fmt.Sprintf("service.port=%d", app.Port), "--set", fmt.Sprintf("containerPort=%d", app.ContainerPort)}
		if options.context != "" {
			helmArgs = append(helmArgs, "--kube-context", options.context)
		}
		return runner.Run(ctx, runner.Command{Name: "helm", Args: helmArgs}, cmd.OutOrStdout(), cmd.ErrOrStderr())
	}
	return command
}

func newStatusCommand(options *cliOptions) *cobra.Command {
	return &cobra.Command{Use: "status [app]", Short: "Show Kubernetes Deployment health", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		app := ""
		if len(args) == 1 {
			app = args[0]
		}
		if app == "" {
			return errors.New("an app name is required")
		}
		client, err := k8s.New(options.kubeconfig, options.context, options.namespace)
		if err != nil {
			return err
		}
		status, err := client.Status(cmd.Context(), app)
		if err != nil {
			return err
		}
		return writeJSON(cmd.OutOrStdout(), status)
	}}
}

func newLogsCommand(options *cliOptions) *cobra.Command {
	var follow bool
	command := &cobra.Command{Use: "logs <app>", Short: "Stream logs from the first application pod", Args: cobra.ExactArgs(1)}
	command.Flags().BoolVarP(&follow, "follow", "f", true, "follow logs")
	command.RunE = func(cmd *cobra.Command, args []string) error {
		client, err := k8s.New(options.kubeconfig, options.context, options.namespace)
		if err != nil {
			return err
		}
		return client.Logs(cmd.Context(), args[0], follow, cmd.OutOrStdout())
	}
	return command
}

func newRollbackCommand(options *cliOptions) *cobra.Command {
	return &cobra.Command{Use: "rollback <app>", Short: "Roll back the Kubernetes Deployment", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := k8s.New(options.kubeconfig, options.context, options.namespace)
		if err != nil {
			return err
		}
		deployment, err := client.Rollback(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "rolled back %s\n", deployment.Name)
		return nil
	}}
}

func newDestroyCommand(options *cliOptions) *cobra.Command {
	return &cobra.Command{Use: "destroy <app>", Short: "Delete an application Helm release", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		destroyArgs := []string{"uninstall", args[0], "--namespace", options.namespace}
		if options.context != "" {
			destroyArgs = append(destroyArgs, "--kube-context", options.context)
		}
		return runner.Run(cmd.Context(), runner.Command{Name: "helm", Args: destroyArgs}, cmd.OutOrStdout(), cmd.ErrOrStderr())
	}}
}

func localRepo(repo string) (string, func(), error) {
	if !isRemote(repo) {
		root, err := config.RepoPath(repo)
		return root, func() {}, err
	}
	temp, err := os.MkdirTemp("", "cloudforge-repo-")
	if err != nil {
		return "", func() {}, err
	}
	if err := runner.Run(context.Background(), runner.Command{Name: "git", Args: []string{"clone", "--depth", "1", repo, temp}}, io.Discard, os.Stderr); err != nil {
		_ = os.RemoveAll(temp)
		return "", func() {}, err
	}
	return temp, func() { _ = os.RemoveAll(temp) }, nil
}

func isRemote(repo string) bool {
	return strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "git@") || strings.HasPrefix(repo, "ssh://")
}

func buildImage(ctx context.Context, root, image string, result detect.Result) error {
	if result.Stack == detect.StackBuildpacks && !detect.HasFile(root, "Dockerfile") {
		return runner.Run(ctx, runner.Command{Name: "pack", Args: []string{"build", image, "--path", root, "--builder", "paketobuildpacks/builder-jammy-base"}}, os.Stdout, os.Stderr)
	}
	return runner.Run(ctx, runner.Command{Name: "docker", Args: []string{"build", "-t", image, root}}, os.Stdout, os.Stderr)
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
