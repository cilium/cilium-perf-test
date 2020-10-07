all: image

image:
	docker build . -t cilium/cilium-perf-test
