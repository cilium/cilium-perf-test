#!/bin/bash

#hackity hack

native_cidr=$(kubectl get pod -l component=kube-proxy -n kube-system -o=jsonpath="{.items[0].spec.containers[0].command}" | grep -oP '\--cluster-cidr=\K[0-9./]*')

namespace=$(kubectl get ns -o name | grep cilium | awk -F "/" '{print $2}')

KUBE_EDITOR="sed -i s;NATIVE_CIDR_PLACEHOLDER;${native_cidr};" kubectl edit cm -n "${namespace}"

kubectl wait -n "${namespace}" --for=condition=Ready --all pod --timeout=10m

/usr/local/bin/perf-test "$@"
