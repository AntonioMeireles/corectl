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

	"github.com/TheNewNormal/corectl/host/session"
	"github.com/TheNewNormal/corectl/server"
	"github.com/spf13/cobra"
)

var (
	killCmd = &cobra.Command{
		Use:     "kill [VMids]",
		Aliases: []string{"stop", "halt"},
		Short:   "Halts one or more running CoreOS instances",
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			session.Caller.CmdLine.BindPFlags(cmd.Flags())
			if len(args) < 1 && !session.Caller.CmdLine.GetBool("all") {
				err = fmt.Errorf("This command requires either at least " +
					"one argument to work or --all.")
			}
			return
		},
		RunE: killCommand,
	}
)

func killCommand(cmd *cobra.Command, args []string) (err error) {
	if _, err = server.Daemon.Running(); err != nil {
		return fmt.Errorf("Cannot connect to the 'corectl' daemon.")
	}

	if session.Caller.CmdLine.GetBool("all") {
		_, err = server.Query("vm:stop", []string{})
	} else {
		_, err = server.Query("vm:stop", args)
	}
	return
}

func init() {
	killCmd.Flags().BoolP("all", "a", false, "halts all running instances")
	rootCmd.AddCommand(killCmd)
}
