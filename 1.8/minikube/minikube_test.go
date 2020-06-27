package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"testing"
	"time"

	kt "github.com/dlespiau/kube-test-harness"
	"github.com/dlespiau/kube-test-harness/logger"
	"github.com/isovalent/hubble-perf/internal/run"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Baseline overhead of running cilium with hubble enabled.
func TestBaseline(t *testing.T) {
	createMinikube(t)

	h := kt.New(kt.Options{
		LogLevel: logger.Debug,
	})
	if err := h.Setup(); err != nil {
		log.Fatal(err)
	}
	test := h.NewTest(t)

	deployCilium(t, test)
	deployMonitoring(t, test)
	exposePrometheus(t, test)

	log.Println("Lettins the cluster run to gather metrics...")
	<-time.After(7 * time.Minute)
	queryCPUMetrics(t, getPrometheusURL(t), 5*time.Minute)

	deleteMinikube(t)
}

func createMinikube(t *testing.T) {
	if isMinikubeRunning(t) {
		t.Fatal("minikube is already running. Delte it ant let the test set it up")
	}

	startCNIMinikube(t)
}

func deleteMinikube(t *testing.T) {
	t.Log("Deleting minikube")
	if err := run.Command(
		"minikube",
		"delete",
	); err != nil {
		t.Fatalf("failed to delete minikube: %v", err)
	}
}

func deployCilium(t *testing.T, test *kt.Test) {
	// deploy cilium kitchen sink. testing library doesn't support this kind of
	// an arbitrary file deploy as far as I can tell. it tried to force manifests
	// into specific namespaces.
	if err := run.Command(
		"kubectl",
		"apply", "-f",
		"../manifests/cilium-hubble-metrics-d4415c6fc.yaml",
	); err != nil {
		t.Fatalf("failed to apply cilium manifest: %v", err)
	}

	// wait for kube-system to come up
	if err := test.WaitForPodsReady(
		"kube-system",
		metav1.ListOptions{},
		1,             // all pods are 1/1
		3*time.Minute, // ui usually takes about ~60s so give it some room
	); err != nil {
		t.Fatal("error waiting for pods", err)
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
		1,
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

func queryCPUMetrics(t *testing.T, base string, duration time.Duration) {
	client, err := api.NewClient(api.Config{
		Address: base,
	})
	if err != nil {
		t.Fatal("rrror creating prometheus client", err)
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := v1.Range{
		Start: time.Now().Add(-duration),
		End:   time.Now(),
		Step:  time.Minute,
	}
	result, _, err := v1api.QueryRange(
		ctx,
		// https://github.com/cilium/cilium/blob/v1.8/examples/kubernetes/addons/prometheus/monitoring-example.yaml#L554
		"max(irate(cilium_process_cpu_seconds_total[1m]))*100",
		r,
	)
	if err != nil {
		t.Fatal("error querying Prometheus", err)
	}
	fmt.Printf("Result:\n%v\n", result)
}

func getPrometheusURL(t *testing.T) string {
	cmd := exec.Command(
		"minikube",
		"service",
		"prometheus",
		"--url",
		"-n",
		"cilium-monitoring",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal("failed to get prometheus url", err, string(out))
	}

	return strings.TrimSpace(string(out))
}

func isMinikubeRunning(t *testing.T) bool {
	return exec.Command("minikube", "status").Run() == nil
}

func startCNIMinikube(t *testing.T) {
	t.Log("Starting minikube")
	if err := run.Command(
		"minikube",
		"start",
		"--network-plugin=cni",
	); err != nil {
		t.Fatalf("failed to start minikube: %v", err)
	}
}
