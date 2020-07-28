# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## 2.0.0
### Breaking change

The way how Summary and Histogram metric types has been changed, they are sent
as they come from Prometheus, without adding custom suffixes or calculating
percentiles from the integration side:

- xxx_bucket is ingested as a Counter (converted to a delta rather than
  accumulative). Metrics containing dimension "le" == '+Inf' are also sent, we
  don't omit them.
- xxx_count is passed through as a Counter (converted to a delta rather than
  accumulative).
- xxx_total passed through as a Counter (converted to a delta rather than
  accumulative).
- xxx_sum is passed through as a NR Summary converted to a delta rather than
  accumulative, Min and Max here should be NaN. Its value can be negative.  The
  count field of the summary should be 1.
- For Prometheus summary metrics, we report the quantile as a dimension and we
  don't add a "percentile" dimension.

## 1.5.0
### Changed
- Change the default for the New Relic telemetry emitter delta calculator 
  expiration age and expiration check interval to 5 minutes. These values can 
  now be configured with the following options
  `telemetry_emitter_delta_expiration_age` and
  `telemetry_emitter_delta_expiration_check_interval`. This solves the issue
  with missing counters when the scrape interval takes more than the hard-coded
  30 minutes values that we had previously.
- Use go modules instead of govendor for managing project dependencies.

## 1.4.0
### Changed
- Redact the password from URLs using simple authentication when storing target metadata.

### Fixed
- Encoding of URL parameter is the annotation/label `prometheus.io/path` is fixed.

## 1.3.0
### Changed
- Upgrade the Go Telemetry SDK to version 0.2.0 to fix an issue with NaN values.

## 1.2.2
### Added
- Obfuscate the license key when logging the configuration. Running on debug
  mode prints the configuration object, this included the license key, to
  prevent users from leaking their credentials when sharing their logs for
  troubleshooting, if the license key is used in any kind of to string, print
  or log statement the symbol `****` will be returned.

### Changed
- The transformation located in the `nri-prometheus-cfg` config map of the
  deploy manifest template is commented by default. It's left in the manifest
  as an example on how to use transformations. Installing with the new manifest
  will not filter metrics and everything will be sent to the New Relic platform.

### Fixed
- Fixed a bug that caused newly created pods without a valid podIP to be discovered
  and cached, and not be scraped unless nri-prometheus was restarted.

## 1.2.1
### Fixed
- Skip the check for the `emitter_ca_file` if it's empty.

## 1.2.0
### Added
- Support for HTTP(S) proxy for the emitters via the config options
  `emitter_proxy`, `emitter_ca_file` and `emitter_insecure_skip_verify`.
- Reconnect support when resource watcher connection is dropped.
- Add support for Histograms and Summaries following
  [New Relic's guidelines for higher-level metric abstractions](https://github.com/newrelic/newrelic-exporter-specs/blob/main/Guidelines.md).

### Changed
- Fix and refactor self describing metrics
- Fix how the scrape interval is respected.

## 0.10.3
### Added
- Decorate samples with `k8s.cluster.name`.

### Fixed
- Logic for target discovery was adding nodes even when they were not labelled and required a label to be scraped.
- The `SCRAPE_DURATION` time is respected even when the scrape cycle might finish earlier.

## 0.10.2
### Added
- Support for scraping static targets using (mutual) TLS authentication.

### Changed
- By default do not skip the TLS verification.
- Revamp of the static target configuration:
    - `endpoints` key renamed to `targets`. If you had endpoints configured before, you need to update your
       configuration and restart the scraper.
    - Each item in target can contain one or many urls in the `urls` key. This key is always a list.
    - Each target hosts its own TLS configuration in the `tls_config` key.
    - The TLS configuration contains 3 keys: `ca_file_path`, `cert_file_path` and `key_file_path`
- Rename rules are executed at the end of the transformations pipeline. Now Copy attributes
rules are being executed before it, so the name of metrics being used for matching the source and
the dest match with the original scraped instead of the renamed one.

### Removed
- Static targets cannot be configured via environment variables anymore, as it requires a complex structure now. See
  the _Changed_ section for more information.

## 0.10.1
### Fixed
- The Telemetry Emitter now correctly handles counter metrics.

## 0.10.0 - 2019-07-02
### Added
- Integration benchmark
- `scrape_duration` configuration option to specify the length in time to distribute the
  scraping from the endpoints.
- Ignore rule accepts now exceptions.

### Changed
- Delta calculator memory optimization
- Avoid growing slices when converting from Prometheus metrics
  to our internal DTO by setting the final capacity. This reduces memory allocs.
- Optimized Target Metadata generation
- Distributed targets fetching on time, to avoid memory peaks and big heaps.
- Putting processing rules in contiguous memory to decrease the working set of
  the process.
- Emitters are now responsible for any delta calculation.
- The default emitter now is the Go Telemetry SDK.

### Removed
- Unused `RawMetrics` field from `TargetMetrics` data structure.

## 0.9.0 - 2019-06-12
### Added
- Transformation rule to add static attributes, i.e. a cluster name, through the configuration.
- Default attributes to decorate the metrics with: `clusterName`, `integrationVersion` and `integrationName`.
- Mount pprof http endpoints when running in debug mode.
- `targetName` attribute added to the metrics when converting from Prometheus metrics.
- Integration benchmark

### Changed
- Now convert Prometheus metric types to our own types in the fetchers.
- Emitters now receive `Metric`s instead of `Observation`s to avoid coupling with the format of especific emitters.
- Convert Prometheus `counter`s into New Relic `count`s
- Adapter the fetcher to use the worker concurrency pattern instead of starting blocked go routines.

### Removed
- `nrMetricName` and `nrMetricType` aren't added as metric attributes anymore. This is done by the Metric API.
- `metadata` field removed from the `Metric` type.
- `labels` field removed from the `Target` type.
- The parallel emitter has been removed.

### Fixed
- Now listing pods when looking for targets.
- The Kubernetes target retriever now removes targets if their scrape label is removed or not `true` anymore.

## 0.8.2 - 2019-04-17
### Fixed
- StartTimeMs is now generated correctly.

## 0.8.1 - 2019-04-16
### Fixed
- StartTimeMs values are correctly sent as milliseconds.
- Update the Kubernetes retriever to continue watching the objects when there is
  an error listing them.

## 0.8.0 - 2019-03-04
### Changed
- Transformations config format
- Internal refactoring

## 0.7.0 - 2019-02-25
### Added
- Add support for reading config options from a configuration file
- Read processing rules from the config file

### Changed
- Rename URL config option and its default value
- Modify docker images to don't use the root user

## 0.6.0 - 2019-02-22
### Changed
- Added more rename attributes for known exporters

## 0.5.0 - 2019-02-22
### Changed
- Prefix metric attributes that are coming from K8s object labels

### Removed
- Removed "entityName" attribute from all the metrics


## 0.4.0 - 2019-02-22
### Added
- Log error when a request to the metric API is not successful
- Support for K8s clusters using RBAC

### Changed
- Rename the attribute with info about the scraped endpoint

## 0.3.0 - 2019-02-20
### Added
- Decorate metrics from K8s targets with K8s context attributes
- Add option to decorate metrics automatically

### Changed
- Ignore unsupported metric type
- Send Prometheus untyped metrics as gauges
- Renamed the integration

## 0.2.0 - 2019-02-20
### Added
- Emitters: "api" and "stdout"
- Configurable list of static endpoints
- Integration metrics

### Changed
- Refactor: Decoupled data processing pipeline
- v0.3.0 metrics format
- Watch Kubernetes objects

### Removed
- New Relic Infrastructure Agent dependency

## 0.1.0 - 2019-02-13
### Added
- Initial version
