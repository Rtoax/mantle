// Copyright 2016 CoreOS, Inc.
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

package qemu

import (
	"io/ioutil"

	"golang.org/x/crypto/ssh"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/local"
	"github.com/flatcar-linux/mantle/system/exec"
)

type machine struct {
	qc          *Cluster
	id          string
	qemu        exec.Cmd
	netif       *local.Interface
	journal     *platform.Journal
	consolePath string
	console     string
}

func (m *machine) ID() string {
	return m.id
}

func (m *machine) IP() string {
	return m.netif.DHCPv4[0].IP.String()
}

func (m *machine) PrivateIP() string {
	return m.netif.DHCPv4[0].IP.String()
}

func (m *machine) RuntimeConf() platform.RuntimeConfig {
	return m.qc.RuntimeConf()
}

func (m *machine) SSHClient() (*ssh.Client, error) {
	return m.qc.SSHClient(m.IP())
}

func (m *machine) PasswordSSHClient(user string, password string) (*ssh.Client, error) {
	return m.qc.PasswordSSHClient(m.IP(), user, password)
}

func (m *machine) SSH(cmd string) ([]byte, []byte, error) {
	return m.qc.SSH(m, cmd)
}

func (m *machine) Reboot() error {
	return platform.RebootMachine(m, m.journal)
}

func (m *machine) Destroy() {
	if err := m.qemu.Kill(); err != nil {
		plog.Errorf("Error killing instance %v: %v", m.ID(), err)
	}

	m.journal.Destroy()

	if buf, err := ioutil.ReadFile(m.consolePath); err == nil {
		m.console = string(buf)
	} else {
		plog.Errorf("Error reading console for instance %v: %v", m.ID(), err)
	}

	m.qc.DelMach(m)
}

func (m *machine) ConsoleOutput() string {
	return m.console
}

func (m *machine) JournalOutput() string {
	if m.journal == nil {
		return ""
	}

	data, err := m.journal.Read()
	if err != nil {
		plog.Errorf("Reading journal for instance %v: %v", m.ID(), err)
	}
	return string(data)
}

func (m *machine) Board() string {
	return m.qc.flight.Options().Board
}
