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
| `deployments.metrics`                                         | Metric file to download                                                                                                                                                                                                | `https://gist.githubusercontent.com/paologallinaharbur/a159ad779ca44fb9f4ff5b006ef475ee/raw/f5d8a5e7350b8d5e1d03f151fa643fb3a02cd07d/Average%2520prom%2520output`                                   |
| `deployments.maxRoutines`                           | Max number of goroutines the prometheus mock server will open (if 0 no limit is imposed)                                                                                                                             | `0`                                  |


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
helm install load ./load-test --values ./load-test/values.yaml -n newrelic
```

Notice that when generating a high number of services it is possible the helm command fails to create/delete all resources leaving an unstable scenario.

To overcome this issue `helm install load ./load-test --values ./load-test/values.yaml -n newrelic | kubectl apply -f -` proved to be more reliable.

## Sample prometheus outputs

Test prometheus metrics, by default the deployments download the average output sample:

 - https://gist.githubusercontent.com/paologallinaharbur/125cca06b5c717503c7672766e3667fe/raw/67070882bee890a9e060189cff1ef316745a652b/Small%2520Prom%2520payload Small Payload 10 Timeseries
 - https://gist.githubusercontent.com/paologallinaharbur/a159ad779ca44fb9f4ff5b006ef475ee/raw/f5d8a5e7350b8d5e1d03f151fa643fb3a02cd07d/Average%2520prom%2520output Average Payload 500 Timeseries
 - https://gist.githubusercontent.com/paologallinaharbur/f03818327921754efc5a997894467ff9/raw/c61168c1d2ea8bde6580144ada6f739fb40a7bbf/Large%2520Prom%2520output Big payload 1000 Timeseries
 

## Compare with real data

To compare the average size of the payload scraped by pomi you can run `SELECT average(nr_stats_metrics_total_timeseries_by_target) FROM Metric where clusterName= 'xxxx' SINCE 30 MINUTES AGO TIMESERIES`$
and get the number of timeseries sent (the average payload here counts 500)

To compare the average time a target takes in order to answer `SELECT average(nr_stats_integration_fetch_target_duration_seconds) FROM Metric where clusterName= 'xxxx'  SINCE 30 MINUTES AGO FACET target LIMIT 500`