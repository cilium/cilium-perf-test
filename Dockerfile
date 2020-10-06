FROM google/cloud-sdk:latest

RUN apt-get update && apt-get install -y apt-transport-https gnupg2 ;\
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - ;\
echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | tee -a /etc/apt/sources.list.d/kubernetes.list ;\
apt-get update ;\
apt-get install -y kubectl golang

ENV GOPATH /go

COPY . /go/src/github.com/cilium/cilium-perf-test
RUN go test -c /go/src/github.com/cilium/cilium-perf-test/1.8/gke -o /usr/local/bin/perf-test
RUN cp /go/src/github.com/cilium/cilium-perf-test/1.8/gke/run_in_test_cluster.sh /usr/local/bin/run_in_test_cluster.sh
