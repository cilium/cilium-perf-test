package main

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	kt "github.com/dlespiau/kube-test-harness"
	"github.com/dlespiau/kube-test-harness/logger"
	"github.com/isovalent/hubble-perf/internal/run"
	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// auth provider for GCP, enables the client to authenticate with GKE without external
	// dependencies (e.g. gcloud CLI)
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const (
	// Make sure this matches the namespace in the Cilium deployment YAML.
	ciliumNamespace = "cilium-perf"
)

// Baseline overhead of running cilium with hubble enabled.
func TestBaseline(t *testing.T) {
	// Assume an already running GKE test cluster
	// TODO: check availability here

	h := kt.New(kt.Options{
		LogLevel: logger.Debug,
	})
	if err := h.Setup(); err != nil {
		log.Fatal(err)
	}
	// h.Setup() above already calls SetKubeconfig, but doesn't handle errors, so we might end
	// up with a harness without a k8s client. So call it to avoid that.
	if err := h.SetKubeconfig(""); err != nil {
		log.Fatal(err)
	}

	test := h.NewTest(t)

	checkPreconditions(t, test, ciliumNamespace)

	// Override namespace, otherwise we would get some random value which doesn't match the
	// manifest.
	test.Namespace = ciliumNamespace
	test.Setup()
	defer test.Close()

	deployCilium(t, test, ciliumNamespace)
	deployMonitoring(t, test)
	exposePrometheus(t, test)

	runTime := 7 * time.Minute
	log.Printf("Letting the cluster run for %v to gather metrics...", runTime)
	<-time.After(runTime)
	queryMetrics(t, getPrometheusURL(t, test), 5*time.Minute)
}

func checkPreconditions(t *testing.T, test *kt.Test, namespace string) {
	if namespace == "kube-system" {
		t.Fatal("Cilium won't run in kube-system namespace on GKE.")
	}

	ns, err := test.GetNamespace(ciliumNamespace)
	switch {
	case apierrors.IsNotFound(err):
	case err != nil:
		t.Fatalf("failed to get namespace %s: %v", ciliumNamespace, err)
	case err == nil && ns != nil:
		t.Fatalf("namespace %s already exists", ciliumNamespace)
	}
}

func deployCilium(t *testing.T, test *kt.Test, namespace string) {
	// deploy cilium kitchen sink. testing library doesn't support this kind of
	// an arbitrary file deploy as far as I can tell. it tried to force manifests
	// into specific namespaces.
	if err := run.Command(
		"kubectl",
		"apply",
		"-n", namespace,
		"-f", "../manifests/cilium-hubble-metrics-gke-de838c984dfd.yaml",
	); err != nil {
		t.Fatalf("failed to apply cilium manifest: %v", err)
	}

	var nodes *corev1.NodeList
	if nodes = test.ListNodes(metav1.ListOptions{}); nodes == nil {
		t.Fatal("error listing nodes")
	}
	// number of cilium daemonsets + cilium-node-init daemonsets +
	// cilium-operator deployment
	numPods := (len(nodes.Items) * 2) + 1

	// wait for pods to come up
	if err := test.WaitForPodsReady(
		namespace,
		metav1.ListOptions{},
		numPods,       // all pods are 1/1
		3*time.Minute, // ui usually takes about ~60s so give it some room
	); err != nil {
		t.Fatal("error waiting for pods", err)
	}

	// restart metrics-server by deleting it so that it's managed by cilium
	if err := run.Command(
		"kubectl",
		"delete",
		"-n", "kube-system",
		"pod",
		"-l", "k8s-app=metrics-server",
	); err != nil {
		t.Fatalf("failed to deploy cilium monitoring: %v", err)
	}

	// wait for metrics-server pod
	if err := test.WaitForPodsReady(
		"kube-system",
		metav1.ListOptions{
			LabelSelector: "k8s-app=metrics-server",
		},
		1, // all pods are 1/1
		2*time.Minute,
	); err != nil {
		t.Fatal("error waiting for metrics-server pod", err)
	}
}

func deployMonitoring(t *testing.T, test *kt.Test) {
	if err := run.Command(
		"kubectl",
		"apply", "-f",
		"../manifests/cilium-monitoring-263ebed.yaml",
	); err != nil {
		t.Fatalf("failed to deploy cilium monitoring: %v", err)
	}

	if err := test.WaitForPodsReady(
		"cilium-monitoring",
		metav1.ListOptions{},
		2,
		3*time.Minute,
	); err != nil {
		t.Fatal("cilium monitoring not ready", err)
	}
}

func exposePrometheus(t *testing.T, test *kt.Test) {
	if err := run.Command(
		"kubectl",
		"apply", "-f",
		"../manifests/expose-prometheus.yaml",
	); err != nil {
		t.Fatalf("failed to deploy cilium monitoring: %v", err)
	}
}

func queryMetrics(t *testing.T, base string, duration time.Duration) {
	client, err := prometheusapi.NewClient(prometheusapi.Config{
		Address: base,
	})
	if err != nil {
		t.Fatal("error creating prometheus client", err)
	}

	promv1api := prometheusv1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := prometheusv1.Range{
		Start: time.Now().Add(-duration),
		End:   time.Now(),
		Step:  time.Minute,
	}

	metrics := []string{
		// https://github.com/cilium/cilium/blob/v1.8/examples/kubernetes/addons/prometheus/monitoring-example.yaml#L554
		"max(irate(cilium_process_cpu_seconds_total[1m]))*100",
		// https://github.com/cilium/cilium/blob/v1.8/examples/kubernetes/addons/prometheus/monitoring-example.yaml#L694
		"max(cilium_process_virtual_memory_bytes{k8s_app=\"cilium\"})",
	}

	fmt.Printf("Results:\n")
	for _, m := range metrics {
		result, _, err := promv1api.QueryRange(
			ctx,
			m,
			r,
		)
		if err != nil {
			t.Fatal("error querying Prometheus", err)
		}
		fmt.Printf("%v\n", result)
	}
}

func getPrometheusURL(t *testing.T, test *kt.Test) string {
	svc := test.GetService("cilium-monitoring", "prometheus")
	nodePort := svc.Spec.Ports[0].NodePort

	var nodes *corev1.NodeList
	if nodes = test.ListNodes(metav1.ListOptions{}); nodes == nil {
		t.Fatal("error listing nodes")
	}

	var externalIP string
	for _, n := range nodes.Items {
		for _, ip := range n.Status.Addresses {
			if ip.Type == corev1.NodeExternalIP {
				externalIP = ip.Address
				break
			}
		}
	}

	if externalIP == "" {
		t.Fatal("could not find node external IP")
	}

	return fmt.Sprintf("http://%s:%d", externalIP, nodePort)
}
