// Copyright 2017 CoreOS, Inc.
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

package torcx

import (
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	// Regression test for https://github.com/coreos/bugs/issues/2079
	// Note: it would be preferable to not conflate docker + torcx in this
	// testing, but rather to use a standalone torcx package/profile
	register.Register(&register.Test{
		Run:         torcxEnable,
		ClusterSize: 1,
		Name:        "torcx.enable-service",
		UserData: conf.ContainerLinuxConfig(`
systemd:
  units:
  - name: docker.service
    enable: true
`),
		Distros: []string{"cl"},

		// https://github.com/coreos/mantle/issues/999
		// On the qemu-unpriv platform the DHCP provides no data, pre-systemd 241 the DHCP server sending
		// no routes to the link to spin in the configuring state. docker.service pulls in the network-online
		// target which causes the basic machine checks to fail
		ExcludePlatforms: []string{"qemu-unpriv"},
	})
}

func torcxEnable(c cluster.TestCluster) {
	m := c.Machines()[0]
	output := c.MustSSH(m, `systemctl is-enabled docker`)
	if string(output) != "enabled" {
		c.Errorf("expected enabled, got %v", output)
	}
}
