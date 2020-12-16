## LOAD TEST

This folder contains the chart, the script and the go-test used by the load-test gh pipeline.

You can also run manually the load tests against a local minikube.

es from the repo root folder:
```bash
minikube --memory 8192 --cpus 4 start
source ./load-test/laod_test.sh
runAllLoadTest
```

The image is compile, deployed with `Skaffold`, the load test chart is deployed with 800 targets and the results from the
prometheus output are collected and parsed with a golang help tool.

Check load_test.sh to gather more information regarding the behaviour.
