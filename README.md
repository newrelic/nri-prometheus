[![Community Project header](https://github.com/newrelic/opensource-website/raw/master/src/images/categories/Community_Project.png)](https://opensource.newrelic.com/oss-category/#community-project)

# New Relic Prometheus OpenMetrics Integration

[![Build Status](https://travis-ci.org/newrelic/nri-prometheus.svg?branch=main)](https://travis-ci.org/newrelic/nri-prometheus.svg?branch=main)
[![CLA assistant](https://cla-assistant.io/readme/badge/newrelic/nri-prometheus)](https://cla-assistant.io/newrelic/nri-prometheus)

Fetch metrics in the Prometheus metrics format, inside or outside Kubernetes, and send them to the New Relic Metrics platform.

## How to use it?

For documentation about how to use it please refer to [New Relic's documentation website](https://docs.newrelic.com/docs/new-relic-prometheus-openmetrics-integration-kubernetes).

Find out more about Prometheus and New Relic in [this blog post](https://blog.newrelic.com/product-news/how-to-monitor-prometheus-metrics/). 

## Development

This integration requires having a Kubernetes cluster available to deploy & run
it. For development, we recommend using [Docker](https://docs.docker.com/install/), [Minikube](https://minikube.sigs.k8s.io/docs/start/) & [skaffold](https://skaffold.dev/docs/getting-started/#installing-skaffold).

However, at the moment the tests are totally isolated and you don't need a cluster to run them.

### Prerequisites

1. **Go 1.13**. This project uses the [error
   wrapping](https://golang.org/doc/go1.13#error_wrapping) support, which makes
   it incompatible with previous Go versions.

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

### Running the Kubernetes Target Retriever locally

It can be useful to run the Kubernetes Target Retriever locally against a remote/local cluster to debug the endpoints that are discovered.
The program located in `/cmd/k8s-target-retriever` is made for this.

To run the program,run the following command in your terminal:
```shell script
# ensure your kubectl is configured correcly & against the correct cluster
kubectl config get-contexts
# run the program 
go run cmd/k8s-target-retriever/main.go
``` 

## Testing

To run the tests execute:

```bash
$ make test
```

## Support

Should you need assistance with New Relic products, you are in good hands with several support diagnostic tools and support channels.

> This [troubleshooting framework](https://discuss.newrelic.com/t/troubleshooting-frameworks/108787) steps you through common troubleshooting questions.
> New Relic offers NRDiag, [a client-side diagnostic utility](https://docs.newrelic.com/docs/using-new-relic/cross-product-functions/troubleshooting/new-relic-diagnostics) that automatically detects common problems with New Relic agents. If NRDiag detects a problem, it suggests troubleshooting steps. NRDiag can also automatically attach troubleshooting data to a New Relic Support ticket.
If the issue has been confirmed as a bug or is a Feature request, please file a Github issue.

**Support Channels**

* [New Relic Documentation](https://docs.newrelic.com): Comprehensive guidance for using our platform
* [New Relic Community](https://discuss.newrelic.com): The best place to engage in troubleshooting questions
* [New Relic Developer](https://developer.newrelic.com/): Resources for building a custom observability applications
* [New Relic University](https://learn.newrelic.com/): A range of online training for New Relic users of every level

## Privacy

At New Relic we take your privacy and the security of your information seriously, and are committed to protecting your information. We must emphasize the importance of not sharing personal data in public forums, and ask all users to scrub logs and diagnostic information for sensitive information, whether personal, proprietary, or otherwise.

We define “Personal Data” as any information relating to an identified or identifiable individual, including, for example, your name, phone number, post code or zip code, Device ID, IP address and email address.

Review [New Relic’s General Data Privacy Notice](https://newrelic.com/termsandconditions/privacy) for more information.

## Contributing

We encourage your contributions to improve the Prometheus integration! Keep in mind when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.

If you have any questions, or to execute our corporate CLA, required if your contribution is on behalf of a company,  please drop us an email at opensource@newrelic.com.

**A note about vulnerabilities**

As noted in our [security policy](/SECURITY.md), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

If you would like to contribute to this project, please review [these guidelines](./CONTRIBUTING.md).

To all contributors, we thank you!  Without your contribution, this project would not be what it is today.

## License
nri-prometheus is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.