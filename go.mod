module github.com/newrelic/nri-prometheus

go 1.16

require (
	github.com/newrelic/infra-integrations-sdk/v4 v4.1.0
	github.com/newrelic/newrelic-telemetry-sdk-go v0.7.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.24.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.16.10
	k8s.io/apimachinery v0.16.10
	k8s.io/client-go v0.15.12
)
