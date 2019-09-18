package telemetry

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// Config customizes the behavior of a Harvester.
type Config struct {
	// Client is the http.Client used for making requests. By default it is
	// given a Timeout of 5 seconds.
	Client *http.Client
	// MaxTries is the maximum number of times to attempt to send a payload
	// before dropping it. By default, MaxTries is set to 3.
	MaxTries int
	// RetryBackoff is the amount of time to wait between attempts to send a
	// payload. By default, RetryBackoff is set to 1 second.
	RetryBackoff time.Duration
	// APIKey is required for metrics.
	APIKey string
	// LicenseKey is required for spans.
	LicenseKey string
	// CommonAttributes are the attributes to be applied to all metrics that
	// use this Config. They are not applied to spans.
	CommonAttributes map[string]interface{}
	// HarvestPeriod controls how frequently data will be sent to New Relic.
	// If HarvestPeriod is zero then NewHarvester will not spawn a goroutine
	// to send data and it is incumbent on the consumer to call
	// Harvester.HarvestNow when data should be sent. By default, HarvestPeriod
	// is set to 5 seconds.
	HarvestPeriod time.Duration
	// ErrorLogger receives errors that occur in this sdk.
	ErrorLogger func(map[string]interface{})
	// DebugLogger receives structured debug log messages.
	DebugLogger func(map[string]interface{})
	// MetricsURLOverride overrides the metrics endpoint if not not empty.
	MetricsURLOverride string
	// SpansURLOverride overrides the spans endpoint if not not empty.
	SpansURLOverride string
	// BeforeHarvestFunc is a callback function that will be called before a
	// harvest occurs.
	BeforeHarvestFunc func(*Harvester)
}

// ConfigAPIKey sets the Config's APIKey which is required for metrics.
func ConfigAPIKey(key string) func(*Config) {
	return func(cfg *Config) {
		cfg.APIKey = key
	}
}

// ConfigLicenseKey sets the Config's LicenseKey which is required for spans.
func ConfigLicenseKey(key string) func(*Config) {
	return func(cfg *Config) {
		cfg.LicenseKey = key
	}
}

// ConfigCommonAttributes adds the given attributes to the Config's
// CommonAttributes.
func ConfigCommonAttributes(attributes map[string]interface{}) func(*Config) {
	return func(cfg *Config) {
		cfg.CommonAttributes = attributes
	}
}

// ConfigHarvestPeriod sets the Config's HarvestPeriod field which controls the
// rate data is reported to New Relic.  If it is set to zero then the Harvester
// will never report data unless HarvestNow is called.
func ConfigHarvestPeriod(period time.Duration) func(*Config) {
	return func(cfg *Config) {
		cfg.HarvestPeriod = period
	}
}

func newBasicLogger(w io.Writer) func(map[string]interface{}) {
	flags := log.Ldate | log.Ltime | log.Lmicroseconds
	lg := log.New(w, "", flags)
	return func(fields map[string]interface{}) {
		if js, err := json.Marshal(fields); nil != err {
			lg.Println(err.Error())
		} else {
			lg.Println(string(js))
		}
	}
}

// ConfigBasicErrorLogger sets the error logger to a simple logger that logs
// to the writer provided.
func ConfigBasicErrorLogger(w io.Writer) func(*Config) {
	return func(cfg *Config) {
		cfg.ErrorLogger = newBasicLogger(w)
	}
}

// ConfigBasicDebugLogger sets the debug logger to a simple logger that logs
// to the writer provided.
func ConfigBasicDebugLogger(w io.Writer) func(*Config) {
	return func(cfg *Config) {
		cfg.DebugLogger = newBasicLogger(w)
	}
}

// configTesting is the config function to be used when testing. It sets the
// APIKey and LicenseKey but disables the harvest goroutine.
func configTesting(cfg *Config) {
	cfg.APIKey = "api-key"
	cfg.LicenseKey = "license-key"
	cfg.HarvestPeriod = 0
}

func (cfg *Config) logError(fields map[string]interface{}) {
	if nil == cfg.ErrorLogger {
		return
	}
	cfg.ErrorLogger(fields)
}

func (cfg *Config) logDebug(fields map[string]interface{}) {
	if nil == cfg.DebugLogger {
		return
	}
	cfg.DebugLogger(fields)
}
