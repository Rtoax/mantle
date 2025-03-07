// Copyright 2018 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"encoding/json"
	"fmt"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/platform"
)

// rpmOstreeDeployment represents some of the data of an rpm-ostree deployment
type rpmOstreeDeployment struct {
	Booted            bool     `json:"booted"`
	Checksum          string   `json:"checksum"`
	Origin            string   `json:"origin"`
	Osname            string   `json:"osname"`
	Packages          []string `json:"packages"`
	RequestedPackages []string `json:"requested-packages"`
	Timestamp         int64    `json:"timestamp"`
	Unlocked          string   `json:"unlocked"`
	Version           string   `json:"version"`
}

// simplifiedRpmOstreeStatus contains deployments from rpm-ostree status
type simplifiedRpmOstreeStatus struct {
	Deployments []rpmOstreeDeployment
}

// GetRpmOstreeStatusJSON returns an unmarshal'ed JSON object that contains
// a limited representation of the output of `rpm-ostree status --json`
func GetRpmOstreeStatusJSON(c cluster.TestCluster, m platform.Machine) (simplifiedRpmOstreeStatus, error) {
	target := simplifiedRpmOstreeStatus{}
	rpmOstreeJSON, err := c.SSH(m, "rpm-ostree status --json")
	if err != nil {
		return target, fmt.Errorf("Could not get rpm-ostree status: %v", err)
	}

	err = json.Unmarshal(rpmOstreeJSON, &target)
	if err != nil {
		return target, fmt.Errorf("Couldn't umarshal the rpm-ostree status JSON data: %v", err)
	}

	return target, nil
}
