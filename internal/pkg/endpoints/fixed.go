// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package endpoints

import "fmt"

type fixedRetriever struct {
	targets []Target
}

// TargetConfig is used to parse endpoints from the configuration file.
type TargetConfig struct {
	Description string
	URLs        []string  `mapstructure:"urls"`
	TLSConfig   TLSConfig `mapstructure:"tls_config"`
}

// TLSConfig is used to store all the configuration required to use Mutual TLS authentication.
type TLSConfig struct {
	CaFilePath         string `mapstructure:"ca_file_path"`
	CertFilePath       string `mapstructure:"cert_file_path"`
	KeyFilePath        string `mapstructure:"key_file_path"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

// FixedRetriever creates a TargetRetriver that returns the targets belonging to the URLs passed as arguments
func FixedRetriever(targetCfgs ...TargetConfig) (TargetRetriever, error) {
	fixed := make([]Target, 0, len(targetCfgs))
	for _, targetCfg := range targetCfgs {
		targets, err := EndpointToTarget(targetCfg)
		if err != nil {
			return nil, fmt.Errorf("parsing target %v: %v", targetCfg, err.Error())
		}
		fixed = append(fixed, targets...)
	}
	return &fixedRetriever{targets: fixed}, nil
}

func (f fixedRetriever) GetTargets() ([]Target, error) {
	return f.targets, nil
}

func (f fixedRetriever) Watch() error {
	// NOOP
	return nil
}

func (f fixedRetriever) Name() string {
	return "fixed"
}
