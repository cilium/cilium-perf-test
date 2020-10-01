# GKE

Use a development GKE cluster to run nightly tests.

## Setup

Create a cluster using the `gcloud` CLI tool from the [Google Cloud
SDK](https://cloud.google.com/sdk/install):

```
# Store project name, zone and cluster name in environment variables.
# More zones can be found at https://cloud.google.com/compute/docs/regions-zones?hl=en_US#available
export GKE_PROJECT=cilium-dev
export GKE_ZONE=europe-west4-a
export GKE_CLUSTER_NAME=your-perf-test-cluster-name

# Set the GCP project in which your cluster will live
gcloud config set project $GKE_PROJECT
# List the current clusters and check that the cluster $GKE_CLUSTER_NAME doesn't exist yet,
# i.e. the command should return no result
gcloud container clusters list | grep $GKE_CLUSTER_NAME

# Set a different kubeconfig to use, if you want
# You can use the default KUBECONFIG but you may need to select other k8s clusters manually to use them
export KUBECONFIG=$HOME/.kube/config

# Create cluster
gcloud container clusters create \
	--release-channel rapid \
	--zone $GKE_ZONE \
	--image-type COS \
	--machine-type n1-standard-4 \
	--num-nodes 2 \
	$GKE_CLUSTER_NAME

# Pull down the k8s credentials for this cluster
gcloud container clusters get-credentials --zone $GKE_ZONE $GKE_CLUSTER_NAME

# Extract the Cluster CIDR to enable native routing:
export GKE_NATIVE_CIDR="$(gcloud container clusters describe $GKE_CLUSTER_NAME --zone $GKE_ZONE --format 'value(clusterIpv4Cidr)')"

# Extract the main firewall rule and update it to allow all IP addresses to
# allow ingress from our host machines for Prometheus:
export GKE_FW_RULE_NAME="$(gcloud compute firewall-rules list --filter "name~'gke-${GKE_CLUSTER_NAME}.+-all'" --format "value(name)")"
gcloud compute firewall-rules update "${GKE_FW_RULE_NAME}" --source-ranges "0.0.0.0/0"
```

Alternativley, you can run `make provision` to create the cluster, pull down
the k8s credentials and extract the firewall rules.

In case you previously resized the cluster (see [Teardown](#teardown) below),
you can scale it up again instead of creating a new cluster:

```
gcloud container clusters resize $GKE_CLUSTER_NAME --node-pool default-pool --num-nodes 2 --zone $GKE_ZONE
```

## Run Tests

Just run `make run` from within this directory. Make sure to keep the environment variables from the previous step in place.

```
make run
```

## Teardown

After you're done, resize or delete the cluster (resizing to 0 is possible, scaled down clusters cost nothing):

```
# Resize cluster to 0 nodes...
gcloud container clusters resize $GKE_CLUSTER_NAME --node-pool default-pool --num-nodes 0 --zone $GKE_ZONE
# ...or delete the cluster
gcloud container clusters delete --zone $GKE_ZONE $GKE_CLUSTER_NAME
```

## Update manifests

Set variables

```
export MANIFESTS=$(pwd)/../manifests
# in $GOPATH/src/github.com/cilium/cilium/install/kubernetes
export GIT_SHA=$(git rev-parse --short HEAD)
```

For Cilium 1.8:

```
helm repo add cilium https://helm.cilium.io/
helm template cilium cilium/cilium \
	--namespace cilium-perf \
	--set global.nodeinit.enabled=true \
	--set nodeinit.reconfigureKubelet=true \
	--set nodeinit.removeCbrBridge=true \
	--set global.cni.binPath=/home/kubernetes/bin \
	--set global.gke.enabled=true \
	--set config.ipam=kubernetes \
	--set nodeinit.restartPods=true \
	--set global.nativeRoutingCIDR=$GKE_NATIVE_CIDR \
	--set global.hubble.enabled=true \
	--set global.hubble.metrics.enabled="{dns,drop,tcp,flow,port-distribution,icmp,http}" \
	--set global.prometheus.enabled=true \
	--set global.operatorPrometheus.enabled=true > ../manifests/cilium-hubble-metrics-gke.yaml

For Cilium latest:

```
helm template cilium \
	--namespace cilium-perf \
	--set global.nodeinit.enabled=true \
	--set nodeinit.reconfigureKubelet=true \
	--set nodeinit.removeCbrBridge=true \
	--set global.cni.binPath=/home/kubernetes/bin \
	--set global.gke.enabled=true \
	--set config.ipam=kubernetes \
	--set global.nativeRoutingCIDR=$GKE_NATIVE_CIDR \
	--set global.hubble.enabled=true \
	--set global.hubble.metrics.enabled="{dns,drop,tcp,flow,port-distribution,icmp,http}" \
	--set global.prometheus.enabled=true \
	--set global.operatorPrometheus.enabled=true > $MANIFESTS/cilium-hubble-metrics-gke-$GIT_SHA.yaml
```
