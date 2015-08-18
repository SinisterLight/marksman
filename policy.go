// Copyright 2015 CodeIgnition. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"time"
)

// PolicyType denotes the monitoring policy type.
//
// e.g. "tcp"
type PolicyType string

// Policy is the map containing the rules of a particular monitoring policy.
// All policies require an "alias" key. The "alias" can be any string to denote the policy,
// like "recon-port-check".
//
// e.g. "tcp" PolicyType requires 2 policy keys "port" and "frequency" along with "alias"
type Policy map[string]string

// PolicyConfig is the format used to encode/decode the monitoring policy
// received from the message queue or to store in the config file
type PolicyConfig map[PolicyType][]Policy

// PolicyFuncMap maps a PolicyType to a handler function
var PolicyFuncMap = map[PolicyType]func(Policy) error{
	"tcp": tcpPolicyHandler,
}

func tcpPolicyHandler(p Policy) error {
	// Always use v, ok := p[key] form to avoid panic
	port, ok := p["port"]
	if !ok {
		return errors.New(`"port" key missing in tcp policy`)
	}
	freq, ok := p["frequency"]
	if !ok {
		return errors.New(`"frequency" key missing in tcp policy`)
	}

	// From the time package docs:
	//
	// ParseDuration parses a duration string.
	// A duration string is a possibly signed sequence of
	// decimal numbers, each with optional fraction and a unit suffix,
	// such as "300ms", "-1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	d, err := time.ParseDuration(freq)
	if err != nil {
		return err
	}

	// This check is here to ensure time.Ticker(d) doesn't panic
	if d <= 0 {
		return errors.New("frequency must be a positive quantity")
	}

	for now := range time.Ticker(d) {
		_, err := net.DialTimeOut("tcp", port, d)
		if err != nil {
			// TODO: sendErrorToMarksman(Agent, Policy, err)
		} else {
			// TODO: sendSuccessToMarksman(Agent, Policy, err)
		}

	}
}
