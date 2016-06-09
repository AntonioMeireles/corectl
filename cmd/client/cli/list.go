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
	"encoding/json"
	"fmt"

	"github.com/TheNewNormal/corectl/host/session"
	"github.com/TheNewNormal/corectl/release/cli"
	"github.com/TheNewNormal/corectl/server"
	"github.com/TheNewNormal/corectl/target/coreos"
	"github.com/blang/semver"
	"github.com/spf13/cobra"
)

var (
	lsCmd = &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "Lists the CoreOS images available locally",
		PreRunE: shared.DefaultPreRunE,
		RunE:    lsCommand,
	}
)

func lsCommand(cmd *cobra.Command, args []string) (err error) {
	var i interface{}
	if _, err = server.Daemon.Running(); err != nil {
		return fmt.Errorf("Cannot connect to the 'corectl' daemon.")
	}

	if i, err = server.Query("images:list", nil); err != nil {
		return
	}
	local := i.(map[string]semver.Versions)
	cli := session.Caller.CmdLine
	channels := []string{coreos.Channel(cli.GetString("channel"))}
	if cli.GetBool("all") {
		channels = coreos.Channels
	}
	if cli.GetBool("json") {
		var pp []byte
		if len(channels) == 1 {
			if pp, err = json.MarshalIndent(
				local[coreos.Channel(cli.GetString("channel"))],
				"", "    "); err != nil {
				return
			}
		} else {
			if pp, err = json.MarshalIndent(local, "", "    "); err != nil {
				return
			}
		}
		fmt.Println(string(pp))
		return
	}
	fmt.Println("locally available images")
	for _, i := range channels {
		var header bool
		for _, d := range local[i] {
			if !header {
				fmt.Printf("  - %s channel \n", i)
				header = true
			}
			fmt.Println("    -", d.String())
		}
	}
	return
}

func init() {
	lsCmd.Flags().String("channel", "alpha", "CoreOS channel")
	lsCmd.Flags().BoolP("all", "a", false, "browses all channels")
	lsCmd.Flags().BoolP("json", "j", false,
		"outputs in JSON for easy 3rd party integration")
	rootCmd.AddCommand(lsCmd)
}
