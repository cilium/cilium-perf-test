module github.com/isovalent/hubble-perf

go 1.14

require (
	github.com/dlespiau/kube-test-harness v0.0.0-20190930170435-ec3f93e1a754
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/stretchr/testify v1.5.1 // indirect
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550 // indirect
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e // indirect
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a // indirect
	golang.org/x/text v0.3.2 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	k8s.io/apimachinery v0.0.0-20180904193909-def12e63c512
)

replace github.com/dlespiau/kube-test-harness => github.com/isovalent/kube-test-harness v0.0.0-20200626171327-922e441c779b
