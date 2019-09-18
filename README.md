# New Relic Prometheus OpenMetrics Integration

Fetch metrics in the Prometheus metrics inside or outside Kubernetes and send them to the New Relic Metrics platform.

## How to use it?

For documentation about how to use it please refer to [New Relic's documentation website](https://docs.newrelic.com/docs/new-relic-prometheus-openmetrics-integration-kubernetes).

## Development

This integration requires having a Kubernetes cluster available to deploy & run
it. For development, we recommend using [Docker](https://docs.docker.com/install/), [Minikube](https://minikube.sigs.k8s.io/docs/start/) & [skaffold](https://skaffold.dev/docs/getting-started/#installing-skaffold).

However, at the moment the tests are totally isolated and you don't need a cluster to run them.

We're currently supporting **Go 1.13**.

### Execute tests

You can do it running `make test`.

### Executing the integration in a development cluster

- You need to configure how to deploy the integration in the cluster. Copy
deploy/local.yaml.example to deploy/local.yaml and edit the placeholders.
 - To get the Infrastructure License key, visit:
   `https://newrelic.com/accounts/<YOUR_ACCOUNT_ID>`. It's located in the right sidebar.
- After updating the yaml file, you need to compile the integration: `GOOS=linux make compile-only`.
- Once you have it compiled, you need to deploy it in your K8s cluster: `skaffold run`

## Release proccess

First, run `release.sh` script with the new version that will be released. This should update the CHANGELOG, the version stored in the code and the manifest that gets uploaded to the download site.

Update the `CHANGELOG.md` file in this repository and create a [GH release](https://github.com/newrelic/nri-prometheus/releases/new).
Use the version of the release as input for the CI Job later on.

Create a Github release for the version that is about to be released. The title of the release should follow the template: `v0.0.0`. The changelog of each version should be part of the release description.

Trigger the [CI Release Job](#pending-link) to build and push the docker image, and to upload the Kubernetes
manifest template to download.newrelic.com.
