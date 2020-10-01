.PHONY: run provision list check-env
run:
	go test -v . -count=1

provision: check-env
	# Create cluster.
	gcloud container clusters create \
		--release-channel rapid \
		--zone $(GKE_ZONE) \
		--image-type COS \
		--machine-type n1-standard-4 \
		--num-nodes 2 \
		$(GKE_CLUSTER_NAME)

	# Pull down the k8s credentials for this cluster.
	gcloud container clusters get-credentials \
		--zone $(GKE_ZONE) \
		$(GKE_CLUSTER_NAME)

	# Extract the main firewall rule and update it to allow all IP addresses to
	# allow ingress from our host machines for Prometheus:
	$(eval GKE_FW_RULE_NAME := $(shell gcloud compute firewall-rules list --filter "name~'gke-$(GKE_CLUSTER_NAME).+-all'" --format "value(name)"))
	gcloud compute firewall-rules update "$(GKE_FW_RULE_NAME)" --source-ranges "0.0.0.0/0"

	@echo "Consider setting the KUBECONFIG var, or select the appropriate context in kubectl."

cleanup:
	-kubectl delete clusterroles cilium cilium-operator prometheus
	-kubectl delete clusterrolebindings cilium cilium-operator prometheus
	-kubectl delete namespace cilium-perf cilium-monitoring

list:
	# List the current clusters.
	gcloud container clusters list

check-env:
# The following lines need to be dedented for Make to pick them up.
ifndef GKE_ZONE
	$(error GKE_ZONE is not set")
endif

ifndef GKE_CLUSTER_NAME
	$(error GKE_CLUSTER_NAME is not set")
endif

build:
	go test -c
