.PHONY: run
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

	@echo "Consider setting the KUBECONFIG var, or select the appropriate context in kubectl."

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
