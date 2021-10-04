module github.com/newrelic/nri-prometheus

go 1.16

require (
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/newrelic/infra-integrations-sdk/v4 v4.1.0
	github.com/newrelic/newrelic-telemetry-sdk-go v0.8.1
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.15.0 // indirect
	github.com/onsi/gomega v1.10.5 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.31.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
)

// To mitigate Snyk security scan.
replace github.com/pkg/sftp => github.com/pkg/sftp v1.11.0
