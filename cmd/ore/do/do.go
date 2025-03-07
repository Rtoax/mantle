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

package do

import (
	"context"
	"fmt"

	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/auth"
	"github.com/flatcar-linux/mantle/cli"
	"github.com/flatcar-linux/mantle/platform/api/do"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "ore/do")

	DO = &cobra.Command{
		Use:   "do [command]",
		Short: "DigitalOcean machine utilities",
	}

	API     *do.API
	options do.Options

	imageName string
	imageURL  string
)

func init() {
	DO.PersistentFlags().StringVar(&options.ConfigPath, "config-file", "", "config file (default \"~/"+auth.DOConfigPath+"\")")
	DO.PersistentFlags().StringVar(&options.Profile, "profile", "", "profile (default \"default\")")
	DO.PersistentFlags().StringVar(&options.AccessToken, "token", "", "access token (overrides config file)")
	cli.WrapPreRun(DO, preflightCheck)
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	plog.Debugf("Running DigitalOcean preflight check")
	api, err := do.New(&options)
	if err != nil {
		return fmt.Errorf("could not create DigitalOcean client: %v", err)
	}
	if err := api.PreflightCheck(context.Background()); err != nil {
		return fmt.Errorf("could not complete DigitalOcean preflight check: %v", err)
	}

	plog.Debugf("Preflight check success; we have liftoff")
	API = api
	return nil
}
