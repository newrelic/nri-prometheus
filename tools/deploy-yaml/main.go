// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// This program generates deploy/nri-prometheus.major.yaml and deploy/nri-prometheus.minor.yaml.
// It is run as by goreleaser before build
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/template"
)

func main() {
	// full-version contains the full semver, e.g. "2.2.1"
	fullVersion := flag.String("full-version", "dev", "Full semver for yaml generation")
	// major-minor-version contains the integration version minus the patch, e.g. "2.2"
	majorMinorVersion := flag.String("major-minor-version", "dev", "Semver without the patch")
	// prerelease should be is non-empty if doing a prerelease
	// It is retrieved from $PRERELEASE, which is currently set by Github Actions
	// TODO: Commented out since release workflow is broken. Everything will be released on pre-releases, and nothing
	// will be done on releases. This renders the pre-release logic inside this generator uselees for now.
	//prerelease := flag.String("prerelease", os.Getenv("PRERELEASE"), "Non-empty string if prereleasing")
	prerelease := flag.String("prerelease", "", "Non-empty string if prereleasing")
	flag.Parse()

	tmpl, err := template.ParseFiles("deploy/nri-prometheus.tmpl.yaml")
	if nil != err {
		log.Fatal(err)
	}

	err = os.MkdirAll("target/deploy", os.ModePerm)
	if nil != err {
		log.Fatal(err)
	}

	// Generate yaml containing the full version, including patch number
	// e.g. nri-prometheus-2.2.1.yml pointing to newrelic/nri-prometheus:2.2.1
	writeTemplate(tmpl, *fullVersion, *fullVersion)

	// Skip generating more yamls if we are prereleasing
	if *prerelease != "" {
		return
	}

	// For full releases, generate a yaml ommiting the patch number, pointing to the same docker image
	// e.g. nri-prometheus-2.2.yml pointing to newrelic/nri-prometheus:2.2
	// All docker images are generated by goreleaser
	writeTemplate(tmpl, *majorMinorVersion, *majorMinorVersion)
	// Generate a nri-prometheus-latest.yml pointing to the last full version e.g. newrelic/nri-prometheus:2.2.1
	writeTemplate(tmpl, *fullVersion, "latest")
}

func writeTemplate(tmpl *template.Template, version string, yamlVersion string) {
	bf := bytes.NewBuffer([]byte{})
	err := tmpl.Execute(bf, struct {
		Version string
	}{Version: version})
	if nil != err {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(
		fmt.Sprintf("target/deploy/nri-prometheus-%s.yaml", yamlVersion),
		bf.Bytes(),
		0644)

	if nil != err {
		log.Fatal(err)
	}
}
