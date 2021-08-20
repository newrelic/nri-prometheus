module github.com/newrelic/nri-prometheus

go 1.16

require (
	github.com/golangci/golangci-lint v1.40.1
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/newrelic/infra-integrations-sdk/v4 v4.1.0
	github.com/newrelic/newrelic-telemetry-sdk-go v0.7.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.26.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20210510120150-4163338589ed // indirect
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	sigs.k8s.io/structured-merge-diff/v4 v4.1.1 // indirect
)

// To avoid CVE-2018-16886 triggering a security scan.
replace go.etcd.io/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20190108173120-83c051b701d3
