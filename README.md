# New Relic Prometheus OpenMetrics Integration

Fetch metrics in the Prometheus metrics inside or outside Kubernetes and send them to the New Relic Metrics platform.

## How to use it?

For documentation about how to use it please refer to [New Relic's documentation website](https://docs.newrelic.com/docs/new-relic-prometheus-openmetrics-integration-kubernetes).

## Development

This integration requires having a Kubernetes cluster available to deploy & run
it. For development, we recommend using [Docker](https://docs.docker.com/install/), [Minikube](https://minikube.sigs.k8s.io/docs/start/) & [skaffold](https://skaffold.dev/docs/getting-started/#installing-skaffold).

However, at the moment the tests are totally isolated and you don't need a cluster to run them.

We're currently supporting **Go 1.13**.

### Running the tests & linters

You can run the linters with `make validate` and the tests with `make test`.

### Build the binary

To build the project run: `make build`. This will output the binary release at `bin/nri-prometheus`.

### Build the docker image

In case you wish to push your own version of the image to a Docker registry, you can build it with:

```bash
IMAGE_NAME=<YOUR_IMAGE_NAME> make docker-build
```

And push it later with `docker push`

### Executing the integration in a development cluster

- You need to configure how to deploy the integration in the cluster. Copy
deploy/local.yaml.example to deploy/local.yaml and edit the placeholders.
 - To get the Infrastructure License key, visit:
   `https://newrelic.com/accounts/<YOUR_ACCOUNT_ID>`. It's located in the right sidebar.
- After updating the yaml file, you need to compile the integration: `GOOS=linux make compile-only`.
- Once you have it compiled, you need to deploy it in your Kubernetes cluster: `skaffold run`
