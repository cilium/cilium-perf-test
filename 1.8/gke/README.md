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
gcloud container clusters create --image-type COS --machine-type n1-standard-4 --zone $GKE_ZONE $GKE_CLUSTER_NAME --num-nodes 2

# Pull down the k8s credentials for this cluster
gcloud container clusters get-credentials --zone $GKE_ZONE $GKE_CLUSTER_NAME
```

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
