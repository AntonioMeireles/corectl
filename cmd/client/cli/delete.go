// Copyright (c) 2016 by António Meireles  <antonio.meireles@reformi.st>.
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
//

package cli

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/helm/helm/log"

	"github.com/TheNewNormal/corectl/host/session"
	"github.com/TheNewNormal/corectl/release/cli"
	"github.com/TheNewNormal/corectl/server"
	"github.com/TheNewNormal/corectl/target/coreos"
	"github.com/spf13/cobra"
)

var (
	rmCmd = &cobra.Command{
		Use:     "rm",
		Aliases: []string{"rmi"},
		Short:   "Remove(s) CoreOS image(s) from the local filesystem",
		PreRunE: shared.DefaultPreRunE,
		RunE:    rmCommand,
	}
)

func rmCommand(cmd *cobra.Command, args []string) (err error) {
	var (
		cli     = session.Caller.CmdLine
		channel = coreos.Channel(cli.GetString("channel"))
		version = coreos.Version(cli.GetString("version"))
		i       interface{}
	)
	if _, err = server.Daemon.Running(); err != nil {
		return fmt.Errorf("Cannot connect to the 'corectl' daemon.")
	}

	if i, err = server.Query("images:list", nil); err != nil {
		return
	}
	local := i.(map[string]semver.Versions)

	l := local[channel]
	if cli.GetBool("old") {
		for _, v := range l[0 : l.Len()-1] {
			if _, err = server.Query("images:remove", []string{channel, v.String()}); err != nil {
				return err
			}
			log.Info("removed %s/%s", channel, v.String())
		}
		return
	}

	if version == "latest" {
		if l.Len() > 0 {
			version = l[l.Len()-1].String()
		} else {
			log.Warn("nothing to delete")
			return
		}
	}
	if _, err = server.Query("images:remove", []string{channel, version}); err != nil {
		return err
	}

	log.Info("removed %s/%s\n", channel, version)

	return
}

func init() {
	rmCmd.Flags().String("channel", "alpha", "CoreOS channel")
	rmCmd.Flags().String("version", "latest", "CoreOS version")
	rmCmd.Flags().Bool("old", false, "removes outdated images")
	rootCmd.AddCommand(rmCmd)
}
