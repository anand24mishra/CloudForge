package k8s

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	client    kubernetes.Interface
	namespace string
}

type Status struct {
	Deployment string      `json:"deployment"`
	Desired    int32       `json:"desired"`
	Ready      int32       `json:"ready"`
	Available  int32       `json:"available"`
	Pods       []PodStatus `json:"pods"`
}

type PodStatus struct {
	Name  string `json:"name"`
	Phase string `json:"phase"`
	Ready bool   `json:"ready"`
}

func New(kubeconfig, contextName, namespace string) (*Client, error) {
	if namespace == "" {
		namespace = "cloudforge"
	}
	loading := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loading.ExplicitPath = kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loading, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("kubeconfig: %w", err)
	}
	return fromConfig(cfg, namespace)
}

func FromConfig(cfg *rest.Config, namespace string) (*Client, error) {
	return fromConfig(cfg, namespace)
}

func fromConfig(cfg *rest.Config, namespace string) (*Client, error) {
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{client: clientset, namespace: namespace}, nil
}

func (c *Client) Status(ctx context.Context, app string) (Status, error) {
	deployment, err := c.client.AppsV1().Deployments(c.namespace).Get(ctx, app, metav1.GetOptions{})
	if err != nil {
		return Status{}, fmt.Errorf("get deployment %s: %w", app, err)
	}
	selector := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: map[string]string{
		"app.kubernetes.io/instance": app,
	}})
	pods, err := c.client.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return Status{}, fmt.Errorf("list pods: %w", err)
	}
	status := Status{Deployment: deployment.Name, Desired: deployment.Status.Replicas, Ready: deployment.Status.ReadyReplicas, Available: deployment.Status.AvailableReplicas, Pods: make([]PodStatus, 0, len(pods.Items))}
	for _, pod := range pods.Items {
		status.Pods = append(status.Pods, PodStatus{Name: pod.Name, Phase: string(pod.Status.Phase), Ready: podReady(&pod)})
	}
	sort.Slice(status.Pods, func(i, j int) bool { return status.Pods[i].Name < status.Pods[j].Name })
	return status, nil
}

func podReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (c *Client) Logs(ctx context.Context, app string, follow bool, out io.Writer) error {
	pods, err := c.client.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/instance=" + app})
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found for %s", app)
	}
	sort.Slice(pods.Items, func(i, j int) bool { return pods.Items[i].Name < pods.Items[j].Name })
	req := c.client.CoreV1().Pods(c.namespace).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{Follow: follow, Timestamps: true})
	stream, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()
	_, err = io.Copy(out, stream)
	return err
}

func (c *Client) Deployment(ctx context.Context, app string) (*appsv1.Deployment, error) {
	return c.client.AppsV1().Deployments(c.namespace).Get(ctx, app, metav1.GetOptions{})
}

// Rollback restores the previous ReplicaSet template by updating the Deployment.
// This is the same revision model used by `kubectl rollout undo`, but remains
// available to the API without shelling out to kubectl.
func (c *Client) Rollback(ctx context.Context, app string) (*appsv1.Deployment, error) {
	deployment, err := c.Deployment(ctx, app)
	if err != nil {
		return nil, err
	}
	currentRevision, _ := strconv.Atoi(deployment.Annotations["deployment.kubernetes.io/revision"])
	selector := metav1.FormatLabelSelector(deployment.Spec.Selector)
	replicaSets, err := c.client.AppsV1().ReplicaSets(c.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("list deployment revisions: %w", err)
	}
	var previous *appsv1.ReplicaSet
	previousRevision := -1
	for i := range replicaSets.Items {
		revision, parseErr := strconv.Atoi(replicaSets.Items[i].Annotations["deployment.kubernetes.io/revision"])
		if parseErr != nil || revision >= currentRevision || revision <= previousRevision {
			continue
		}
		candidate := replicaSets.Items[i]
		previous = &candidate
		previousRevision = revision
	}
	if previous == nil {
		return nil, fmt.Errorf("no previous revision found for deployment %s", app)
	}
	deployment.Spec.Template = *previous.Spec.Template.DeepCopy()
	updated, err := c.client.AppsV1().Deployments(c.namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("update deployment for rollback: %w", err)
	}
	return updated, nil
}

func ParseReadyLine(status Status) string {
	return fmt.Sprintf("%d/%d ready", status.Ready, status.Desired)
}

func SanitizeContext(value string) string {
	return filepath.Base(strings.TrimSpace(value))
}

func ReadLines(r io.Reader) []string {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
