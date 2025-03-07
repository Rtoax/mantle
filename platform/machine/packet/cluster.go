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

package packet

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/packet"
	"github.com/flatcar-linux/mantle/platform/conf"

	"github.com/packethost/packngo"
)

type cluster struct {
	*platform.BaseCluster
	flight   *flight
	sshKeyID string
}

func (pc *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	conf, err := pc.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_PACKET_IPV4_PUBLIC_0}",
		"$private_ipv4": "${COREOS_PACKET_IPV4_PRIVATE_0}",
	})
	if err != nil {
		return nil, err
	}

	vmname := pc.vmname()
	// Stream the console somewhere temporary until we have a machine ID
	consolePath := filepath.Join(pc.RuntimeConf().OutputDir, "console-"+vmname+".txt")
	var cons *console
	var pcons packet.Console // need a nil interface value if unused
	var device *packngo.Device
	// Do not shadow assignments to err (i.e., use a, err := something) in the for loop
	// because the "continue" case needs to access the previous error to return it when the
	// maximal number of retries is reached or to print it at the beginning of the loop.
	for retry := 0; retry <= 2; retry++ {
		if err != nil {
			plog.Warningf("Retrying to provision a machine after error: %q", err)
			if pc.sshKeyID != "" {
				err = os.Remove(consolePath)
				if err != nil && !os.IsNotExist(err) {
					return nil, err
				}
			}
		}
		if pc.sshKeyID != "" {
			// We can only read the console if Packet has our SSH key
			var f *os.File
			f, err = os.OpenFile(consolePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
			if err != nil {
				return nil, err
			}
			cons = &console{
				pc:   pc,
				f:    f,
				done: make(chan interface{}),
			}
			pcons = cons
		}

		// CreateDevice unconditionally closes console when done with it
		device, err = pc.flight.api.CreateDevice(vmname, conf, pcons)
		if err != nil {
			continue // provisioning error
		}

		mach := &machine{
			cluster: pc,
			device:  device,
			console: cons,
		}
		mach.publicIP = pc.flight.api.GetDeviceAddress(device, 4, true)
		mach.privateIP = pc.flight.api.GetDeviceAddress(device, 4, false)
		if mach.publicIP == "" || mach.privateIP == "" {
			mach.Destroy()
			err = fmt.Errorf("couldn't find IP addresses for device")
			continue // provisioning error
		}

		dir := filepath.Join(pc.RuntimeConf().OutputDir, mach.ID())
		if err = os.Mkdir(dir, 0777); err != nil {
			mach.Destroy()
			return nil, err
		}

		if cons != nil {
			if err = os.Rename(consolePath, filepath.Join(dir, "console.txt")); err != nil {
				mach.Destroy()
				return nil, err
			}
		}

		confPath := filepath.Join(dir, "user-data")
		if err = conf.WriteFile(confPath); err != nil {
			mach.Destroy()
			return nil, err
		}

		if mach.journal, err = platform.NewJournal(dir); err != nil {
			mach.Destroy()
			return nil, err
		}

		if err = platform.StartMachine(mach, mach.journal); err != nil {
			mach.Destroy()
			continue // provisioning error
		}

		pc.AddMach(mach)

		return mach, nil

	}

	return nil, err
}

func (pc *cluster) vmname() string {
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", pc.Name()[0:13], b)
}

func (pc *cluster) Destroy() {
	pc.BaseCluster.Destroy()
	pc.flight.DelCluster(pc)
}
