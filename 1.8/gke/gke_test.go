package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	kt "github.com/dlespiau/kube-test-harness"
	"github.com/dlespiau/kube-test-harness/logger"
	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	// auth provider for GCP, enables the client to authenticate with GKE without external
	// dependencies (e.g. gcloud CLI)
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const (
	ciliumNamespace           = "cilium-perf"
	ciliumMonitoringNamespace = "cilium-monitoring"
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

// Reading of YAML file copied from github.com/cilium/cilium/pkg/policy/trace/yaml.go
// TODO: use k8s.io/apimachinery/pkg/util/yaml or similar
func loadYAML(t *testing.T, manifest string) [][]byte {
	f, err := os.Open(manifest)
	if err != nil {
		t.Fatalf("failed to open manifest %q: %s", manifest, err)
	}
	defer f.Close()

	m, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read manifest %q: %s", manifest, err)
	}

	return bytes.Split(m, []byte("---"))
}

// deployManifest deploys all k8s objects defined in the file manifest to namespace.
func deployManifest(t *testing.T, test *kt.Test, manifest, namespace string) {
	docs := loadYAML(t, manifest)
	for _, d := range docs {
		if len(d) < 2 {
			continue
		}

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(d, nil, nil)
		if err != nil {
			t.Log(d)
			t.Fatalf("failed to decode: %s", err)
		}

		switch obj.(type) {
		case *rbacv1.ClusterRole:
			cr := obj.(*rbacv1.ClusterRole)
			test.CreateClusterRole(cr)
		case *rbacv1.ClusterRoleBinding:
			crb := obj.(*rbacv1.ClusterRoleBinding)
			test.CreateClusterRoleBinding(crb)
		case *corev1.ConfigMap:
			cm := obj.(*corev1.ConfigMap)
			test.CreateConfigMap(namespace, cm)
		case *appsv1.DaemonSet:
			ds := obj.(*appsv1.DaemonSet)
			test.CreateDaemonSet(namespace, ds)
		case *appsv1.Deployment:
			d := obj.(*appsv1.Deployment)
			test.CreateDeployment(namespace, d)
		case *corev1.Namespace:
			n := obj.(*corev1.Namespace)
			test.CreateNamespace(n.Name)
		case *corev1.Service:
			s := obj.(*corev1.Service)
			test.CreateService(namespace, s)
		case *corev1.ServiceAccount:
			sa := obj.(*corev1.ServiceAccount)
			test.CreateServiceAccount(namespace, sa)
		default:
			t.Fatalf("k8s resource %T not handled", obj)
		}
	}
}

func deployCilium(t *testing.T, test *kt.Test, namespace string) {
	// deploy cilium kitchen sink
	deployManifest(t, test, "../manifests/cilium-hubble-metrics-gke-de838c984dfd.yaml", test.Namespace)

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
	metricsServerLabel := "k8s-app=metrics-server"
	pods := test.ListPods("kube-system", metav1.ListOptions{
		LabelSelector: metricsServerLabel,
	})
	for _, p := range pods.Items {
		t.Logf("Deleting pod %s matching label %s", p.Name, metricsServerLabel)
		test.DeletePod(&p)
	}

	// wait for metrics-server pod
	if err := test.WaitForPodsReady(
		"kube-system",
		metav1.ListOptions{
			LabelSelector: metricsServerLabel,
		},
		1, // all pods are 1/1
		2*time.Minute,
	); err != nil {
		t.Fatal("error waiting for metrics-server pod", err)
	}
}

func deployMonitoring(t *testing.T, test *kt.Test) {
	deployManifest(t, test, "../manifests/cilium-monitoring-263ebed.yaml", ciliumMonitoringNamespace)

	if err := test.WaitForPodsReady(
		ciliumMonitoringNamespace,
		metav1.ListOptions{},
		2,
		3*time.Minute,
	); err != nil {
		t.Fatal("cilium monitoring not ready", err)
	}
}

func exposePrometheus(t *testing.T, test *kt.Test) {
	docs := loadYAML(t, "../manifests/expose-prometheus.yaml")
	for _, d := range docs {
		if len(d) < 2 {
			continue
		}

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(d, nil, nil)
		if err != nil {
			t.Log(d)
			t.Fatalf("failed to decode: %s", err)
		}

		switch obj.(type) {
		case *corev1.Service:
			newSvc := obj.(*corev1.Service)
			svc := test.GetService(ciliumMonitoringNamespace, newSvc.Name)
			// TODO: for now this just hard-codes the fields the expose-prometheus.xml
			// manifest specifies.
			svc.Spec.Ports = newSvc.Spec.Ports
			svc.Spec.Selector = newSvc.Spec.Selector
			svc.Spec.Type = newSvc.Spec.Type
			test.UpdateService(svc)
		default:
			t.Fatalf("k8s resource %T not handled", obj)
		}
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
