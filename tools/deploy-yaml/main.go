// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// This program generates deploy/nri-prometheus.major.yaml and deploy/nri-prometheus.minor.yaml.
// It can be invoked by running
// go generate
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/template"
)

var (
	majorVersion = "dev"
	minorVersion = "dev"
)

func main() {
	tmpl, err := template.ParseFiles("../../deploy/nri-prometheus.tmpl.yaml")
	if nil != err {
		log.Fatal(err)
	}

	err = os.MkdirAll("../../target/deploy", os.ModePerm)
	if nil != err {
		log.Fatal(err)
	}
	writeTemplate(tmpl, majorVersion, majorVersion)
	writeTemplate(tmpl, minorVersion, minorVersion)
	writeTemplate(tmpl, minorVersion, "latest")
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
		fmt.Sprintf("../../target/deploy/nri-prometheus-%s.yaml", yamlVersion),
		bf.Bytes(),
		0644)

	if nil != err {
		log.Fatal(err)
	}
}
