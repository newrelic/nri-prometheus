This tool allows to generate the relabeling configs needed to override the [metric type mappings](https://docs.newrelic.com/docs/infrastructure/prometheus-integrations/install-configure-remote-write/set-your-prometheus-remote-write-integration#mapping) performed by the New Relic Prometheus Remote Write endpoint.

It process the output of the Prometheus ['/api/v1/metadata'](https://prometheus.io/docs/prometheus/latest/querying/api/#querying-metric-metadata) endpoint and prints the needed relabel configs to stdout

```shell
$ curl -s localhost:9090/api/v1/metadata | ./promconfig
write_relabel_configs:
    - source_labels: '[__name__]'
      regex: ^kubedns_dnsmasq_misses$
      target_label: newrelic_metric_type
      replacement: counter
      action: replace
    - source_labels: '[__name__]'
      regex: ^kubedns_dnsmasq_insertions$
      target_label: newrelic_metric_type
      replacement: counter
      action: replace
```