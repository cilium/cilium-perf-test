# Minikube

So far this is more or less proof of concept and an example for how one can
write a test and get some metrics back.

The metrics query is probably not correct and needs to be adjusted to have
lower resolution and more points that can be reasoned about.

```
❯ make
go test -v . -count=1
=== RUN   TestBaseline
    TestBaseline: minikube_test.go:163: Starting minikube
2020/06/27 16:07:16 Running "minikube start --network-plugin=cni"
* minikube v1.9.2 on Darwin 10.15.4
* Automatically selected the hyperkit driver
* Starting control plane node m01 in cluster minikube
* Creating hyperkit VM (CPUs=2, Memory=4000MB, Disk=20000MB) ...
* Preparing Kubernetes v1.18.0 on Docker 19.03.8 ...
* Enabling addons: default-storageclass, storage-provisioner
* Done! kubectl is now configured to use "minikube"
        harness.go:155: using kubeconfig: /Users/glibsm/.kube/config
    TestBaseline: test.go:68: using API server https://192.168.64.24:8443
2020/06/27 16:07:58 Running "kubectl apply -f ../manifests/cilium-hubble-metrics-d4415c6fc.yaml"
serviceaccount/cilium created
serviceaccount/cilium-operator created
configmap/cilium-config created
clusterrole.rbac.authorization.k8s.io/cilium created
clusterrole.rbac.authorization.k8s.io/cilium-operator created
clusterrolebinding.rbac.authorization.k8s.io/cilium created
clusterrolebinding.rbac.authorization.k8s.io/cilium-operator created
service/hubble-metrics created
daemonset.apps/cilium created
deployment.apps/cilium-operator created
2020/06/27 16:08:06 Running "kubectl apply -f ../manifests/cilium-monitoring-263ebed.yaml"
namespace/cilium-monitoring created
serviceaccount/prometheus-k8s created
configmap/grafana-config created
configmap/grafana-cilium-dashboard created
configmap/grafana-cilium-operator-dashboard created
configmap/grafana-hubble-dashboard created
configmap/prometheus created
clusterrole.rbac.authorization.k8s.io/prometheus created
clusterrolebinding.rbac.authorization.k8s.io/prometheus created
service/grafana created
service/prometheus created
deployment.apps/grafana created
deployment.apps/prometheus created
2020/06/27 16:08:57 Running "kubectl apply -f ../manifests/expose-prometheus.yaml"
service/prometheus configured
2020/06/27 16:08:58 Lettins the cluster run to gather metrics...
Result:
{} =>
1.100000000000001 @[1593299458.451]
0.999900009999001 @[1593299518.451]
1.0000000000000009 @[1593299578.451]
0.999900009999001 @[1593299638.451]
0.999900009999001 @[1593299698.451]
0.9000000000000073 @[1593299758.451]
    TestBaseline: minikube_test.go:52: Deleting minikube
2020/06/27 16:15:58 Running "minikube delete"
* Deleting "minikube" in hyperkit ...
* Removed all traces of the "minikube" cluster.
--- PASS: TestBaseline (525.80s)
PASS
ok      github.com/isovalent/hubble-perf/1.8/minikube   526.125s
```
