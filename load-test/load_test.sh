#!/bin/bash

#Clean away old resources not useful anymore
cleanOldResources(){
  kubectl delete namespace newrelic-load || true
  rm ./load-test/load_test.results || true
}

#Deploy loadTest chart
deployLoadTestEnvironment(){
  kubectl create namespace newrelic-load
  ## we are using the template and not the install since helm suffers when deploying at the same time 800+ resources "http2: stream closed"
  helm template load ./charts/load-test-environment --values ./charts/load-test-environment/values.yaml -n newrelic-load | kubectl apply -f - -n newrelic-load
}

#Compile and deploy with skaffold last version of nri-prometheus
deployCurrentNriPrometheus(){
  # We need to statically link libraries otherwise in the current test Docker image the command could fail
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/nri-prometheus ./cmd/nri-prometheus/
  yq eval '.spec.template.spec.containers[0].env[0].value = env(NEWRELIC_LICENSE)' ./deploy/local.yaml.example > ./deploy/local.yaml
  skaffold run
}

#Retrieve the results of the tests from the prometheus output of the integration
retrieveResults(){
  POD=$(kubectl get pods -n default -l app=nri-prometheus -o jsonpath="{.items[0].metadata.name}")
  kubectl logs ${POD}
  kubectl exec -n default ${POD} -- wget localhost:8080/metrics -q -O - > ./load-test/load_test.results
  # Debug This might be needed when developing locally
  #dos2unix ./load-test/load_test.results
}

#Verify the results of the tests (memory, time elapsed, total targets)
verifyResults(){
  # we need the loadtests flag in order to make sure that these tests are run only needed
  go test -v -tags=loadtests ./load-test/...
}

runLoadTest(){
  if [ -z "$NEWRELIC_LICENSE" ]
  then
    echo "NEWRELIC_LICENSE environment variable should be set"
  else
    cleanOldResources
    deployLoadTestEnvironment
    deployCurrentNriPrometheus
    sleep 180
    retrieveResults
    verifyResults
  fi
}








