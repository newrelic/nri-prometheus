<a href="https://opensource.newrelic.com/oss-category/#community-plus"><picture><source media="(prefers-color-scheme: dark)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/dark/Community_Plus.png"><source media="(prefers-color-scheme: light)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Plus.png"><img alt="New Relic Open Source community plus project banner." src="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Plus.png"></picture></a>

# New Relic Prometheus OpenMetrics integration

> 🚧 Important Notice
> 
> Prometheus Open Metrics integration for Kubernetes has been discontinued and will only be supported until the end of June 2023.
>
> See how to install the [Prometheus agent](https://docs.newrelic.com/docs/infrastructure/prometheus-integrations/install-configure-prometheus-agent/install-prometheus-agent/) to understand its benefits and get a full visibility of your Prometheus workloads running in a Kubernetes cluster.
>
> In case you need to migrate from the Prometheus Open Metrics integration to Open Metrics check the following [migration guide](https://docs.newrelic.com/docs/infrastructure/prometheus-integrations/install-configure-prometheus-agent/migration-guide/).

Fetch metrics in the Prometheus metrics format, inside or outside Kubernetes, and send them to the New Relic platform.

## Installation and usage

For documentation about how to use the integration, refer to [our documentation website](https://docs.newrelic.com/docs/new-relic-prometheus-openmetrics-integration-kubernetes).

Find out more about Prometheus and New Relic in [this blog post](https://blog.newrelic.com/product-news/how-to-monitor-prometheus-metrics/).

## Helm chart

You can install this chart using [`nri-bundle`](https://github.com/newrelic/helm-charts/tree/master/charts/nri-bundle) located in the
[helm-charts repository](https://github.com/newrelic/helm-charts) or directly from this repository by adding this Helm repository:

```shell
helm repo add nri-prometheus https://newrelic.github.io/nri-prometheus
helm upgrade --install nri-prometheus/nri-prometheus -f your-custom-values.yaml
```

For further information of the configuration needed for the chart just read the [chart's README](/charts/nri-prometheus/README.md).

## Building

Golang is required to build the integration. We recommend Golang 1.11 or higher.

This integration requires having a Kubernetes cluster available to deploy and run it. For development, we recommend using [Docker](https://docs.docker.com/install/), [Minikube](https://minikube.sigs.k8s.io/docs/start/), and [skaffold](https://skaffold.dev/docs/getting-started/#installing-skaffold).

After cloning this repository, go to the directory of the Prometheus integration and build it:

```bash
make
```

The command above executes the tests for the Prometheus integration and builds an executable file called `nri-prometheus` under the `bin` directory.

To start the integration, run `nri-prometheus`:

```bash
./bin/nri-prometheus
```

If you want to know more about usage of `./bin/nri-prometheus`, pass the `-help` parameter:

```bash
./bin/nri-prometheus -help
```

External dependencies are managed through the [govendor tool](https://github.com/kardianos/govendor). Locking all external dependencies to a specific version (if possible) into the vendor directory is required.

### Build the Docker image

In case you wish to push your own version of the image to a Docker registry, you can build it with:

```bash
IMAGE_NAME=<YOUR_IMAGE_NAME> make docker-build
```

And push it later with `docker push`

### Executing the integration in a development cluster

- You need to configure how to deploy the integration in the cluster. Copy deploy/local.yaml.example to deploy/local.yaml and edit the placeholders.
- To get the New Relic license key, visit:
   `https://newrelic.com/accounts/<YOUR_ACCOUNT_ID>`. It's located in the right sidebar.
- After updating the yaml file, you need to compile the integration: `GOOS=linux make compile-only`.
- Once you have it compiled, you need to deploy it in your Kubernetes cluster: `skaffold run`

### Running the Kubernetes Target Retriever locally

It can be useful to run the Kubernetes Target Retriever locally against a remote/local cluster to debug the endpoints that are discovered. The binary located in `/cmd/k8s-target-retriever` is made for this.

To run the program, run the following command in your terminal:

```shell script
# ensure your kubectl is configured correcly & against the correct cluster
kubectl config get-contexts
# run the program
go run cmd/k8s-target-retriever/main.go
```

## Testing

To run the tests execute:

```bash
make test
```

At the moment, tests are totally isolated and you don't need a cluster to run them.

## Support

Should you need assistance with New Relic products, you are in good hands with several support diagnostic tools and support channels.

> New Relic offers NRDiag, [a client-side diagnostic utility](https://docs.newrelic.com/docs/using-new-relic/cross-product-functions/troubleshooting/new-relic-diagnostics) that automatically detects common problems with New Relic agents. If NRDiag detects a problem, it suggests troubleshooting steps. NRDiag can also automatically attach troubleshooting data to a New Relic Support ticket.

If the issue has been confirmed as a bug or is a Feature request, please file a Github issue.

**Support Channels**

- [New Relic Documentation](https://docs.newrelic.com): Comprehensive guidance for using our platform
- [New Relic Community](https://forum.newrelic.com): The best place to engage in troubleshooting questions
- [New Relic Developer](https://developer.newrelic.com/): Resources for building a custom observability applications
- [New Relic University](https://learn.newrelic.com/): A range of online training for New Relic users of every level
- [New Relic Technical Support](https://support.newrelic.com/) 24/7/365 ticketed support. Read more about our [Technical Support Offerings](https://docs.newrelic.com/docs/licenses/license-information/general-usage-licenses/support-plan).

## Privacy

At New Relic we take your privacy and the security of your information seriously, and are committed to protecting your information. We must emphasize the importance of not sharing personal data in public forums, and ask all users to scrub logs and diagnostic information for sensitive information, whether personal, proprietary, or otherwise.

We define “Personal Data” as any information relating to an identified or identifiable individual, including, for example, your name, phone number, post code or zip code, Device ID, IP address, and email address.

For more information, review [New Relic’s General Data Privacy Notice](https://newrelic.com/termsandconditions/privacy).

## Contribute

We encourage your contributions to improve this project! Keep in mind that when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.

If you have any questions, or to execute our corporate CLA (which is required if your contribution is on behalf of a company), drop us an email at opensource@newrelic.com.

**A note about vulnerabilities**

As noted in our [security policy](../../security/policy), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

If you would like to contribute to this project, review [these guidelines](./CONTRIBUTING.md).

To all contributors, we thank you!  Without your contribution, this project would not be what it is today.

## License

nri-prometheus is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.
