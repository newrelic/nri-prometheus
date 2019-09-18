// Package telemetry is the recommended way of interacting with the New
// Relic Metrics and Spans HTTP APIs.
//
// This package provides basic interaction with the New Relic Metrics and Spans
// HTTP APIs, automatic batch harvesting on a given schedule, and handling of
// errors from the API response.
//
// To aggregate metrics between harvests, use the instrumentation package.
//
package telemetry
