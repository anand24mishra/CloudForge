package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/anandmishra1/cloudforge/internal/api"
	"github.com/anandmishra1/cloudforge/internal/k8s"
)

func main() {
	listen := flag.String("listen", ":8080", "HTTP listen address")
	namespace := flag.String("namespace", "cloudforge", "Kubernetes namespace")
	kubeconfig := flag.String("kubeconfig", "", "kubeconfig path")
	contextName := flag.String("context", "", "Kubernetes context")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	var kube *k8s.Client
	if client, err := k8s.New(*kubeconfig, *contextName, *namespace); err == nil {
		kube = client
		logger.Info("Kubernetes client configured", "namespace", *namespace)
	} else {
		logger.Warn("Kubernetes client unavailable; generation API remains available", "error", err)
	}

	server := api.NewServer(logger, kube)
	logger.Info("CloudForge API listening", "address", *listen)
	if err := http.ListenAndServe(*listen, server.Handler()); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
