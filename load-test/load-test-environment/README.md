# New Relic's Prometheus Load Generator

## Chart Details

This chart will deploy a prometheus load generator.

## Configuration

| Parameter                                                  | Description                                                                                                                                                                                                                           | Default                                |
|------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------|
| `numberServicesPerDeploy`                                        | Number of services per deployment to create                                                                                                                                                                                                        |                          |
| `deployments.*`                                             | List of specification of the deployments to create                                                                                                                                                                                                      | `[]`                                   |
| `deployments.latency`                                              | time in millisecond the /metric endpoint will wait before answering                                                                                                                                                                        |                `0`                    |
| `deployments.latencyVariation`                                                 | ± latency variation %                                                                                                                                                                                                  | `0`                                   |
| `deployments.metrics`                                         | Metric file to download                                                                                                                                                                                                | Average Load URL*                                 |
| `deployments.maxRoutines`                           | Max number of goroutines the prometheus mock server will open (if 0 no limit is imposed)                                                                                                                             | `0`                                  |

*Average load URL: https://gist.githubusercontent.com/paologallinaharbur/a159ad779ca44fb9f4ff5b006ef475ee/raw/f5d8a5e7350b8d5e1d03f151fa643fb3a02cd07d/Average%2520prom%2520output

## Resources created

Number of targets created `numberServicesPerDeploy * len(deployments)`
Each service has the label `prometheus.io/scrape: "true"` that is automatically detected by nri-prometheus

Resources are generated automatically according the following specifications
 - Name of deployment: `<name>-lat<latency>-latvar<latencyVar>-<deployindex>`
 - Name of service: `<name>-lat<latency>-latvar<latencyVar>-<deployindex>-<serviceindex>`

When increasing the number of targets and the size the error is shown `Request Entity Too Large 413`
Adding in the environment variables of POMI seems to solve it reducing the payload
```
  - name: EMITTER_HARVEST_PERIOD
    value: 200ms
```

## Example

Then, to install this chart, run the following command:

```sh
helm install load ./load-test-environment --values ./load-test-environment/values.yaml -n newrelic
```

Notice that when generating a high number of services it is possible the helm command fails to create/delete all resources leaving an unstable scenario.

To overcome this issue `helm install load ./load-test-environment --values ./load-test-environment/values.yaml -n newrelic | kubectl apply -f -` proved to be more reliable.

## Sample prometheus outputs

Test prometheus metrics, by default the deployments download the big output sample:

 - https://raw.githubusercontent.com/newrelic/nri-prometheus/main/load-test/mockexporter/load_test_small_sample.data Small Payload 10 Timeseries
 - https://raw.githubusercontent.com/newrelic/nri-prometheus/main/load-test/mockexporter/load_test_average_sample.data Average Payload 500 Timeseries
 - https://raw.githubusercontent.com/newrelic/nri-prometheus/main/load-test/mockexporter/load_test_big_sample.data Big payload 1000 Timeseries


## Compare with real data

To compare the average size of the payload scraped by pomi you can run `SELECT average(nr_stats_metrics_total_timeseries_by_target) FROM Metric where clusterName='xxxx' SINCE 30 MINUTES AGO TIMESERIES`$
and get the number of timeseries sent (the average payload here counts 500)

To compare the average time a target takes in order to answer `SELECT average(nr_stats_integration_fetch_target_duration_seconds) FROM Metric where clusterName='xxxx'  SINCE 30 MINUTES AGO FACET target LIMIT 500`
