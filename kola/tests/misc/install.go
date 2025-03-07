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

package misc

import (
	"bytes"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Run:         InstallCloudConfig,
		ClusterSize: 1,
		Name:        "cl.install.cloudinit",
		UserData: conf.Ignition(`{
  "ignition": { "version": "2.0.0" },
  "storage": {
    "files": [{
      "filesystem": "root",
      "path": "/var/lib/flatcar-install/user_data",
      "contents": { "source": "data:,%23cloud-config%0Ahostname:%20%22cloud-config-worked%22" },
      "mode": 420
    }]
  }
}`),
		Distros:          []string{"cl"},
		ExcludePlatforms: []string{"azure"},
	})
}

// Simulate coreos-install features

// Verify that the coreos-install cloud-config path is used
func InstallCloudConfig(c cluster.TestCluster) {
	m := c.Machines()[0]

	// Verify the host name was set from the cloud-config file
	if output, err := c.SSH(m, "hostname"); err != nil || !bytes.Equal(output, []byte("cloud-config-worked")) {
		c.Fatalf("hostname: %q: %v", output, err)
	}
}
