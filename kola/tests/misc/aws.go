// Copyright 2018 CoreOS, Inc.
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
	"fmt"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
)

func init() {
	register.Register(&register.Test{
		Name:        "coreos.misc.aws.diskfriendlyname",
		Platforms:   []string{"aws"},
		Run:         awsVerifyDiskFriendlyName,
		ClusterSize: 1,
		// Previously broken on NVMe devices, see
		// https://github.com/coreos/bugs/issues/2399
		MinVersion: semver.Version{Major: 1828},
		Distros:    []string{"cl", "rhcos"},
	})
}

// Check invariants on AWS instances.

func awsVerifyDiskFriendlyName(c cluster.TestCluster) {
	friendlyName := "/dev/xvda"
	c.MustSSH(c.Machines()[0], fmt.Sprintf("stat %s", friendlyName))
}
