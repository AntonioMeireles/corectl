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
	"github.com/TheNewNormal/corectl/release"
	"github.com/TheNewNormal/corectl/release/cli"
	"github.com/TheNewNormal/corectl/server"
	"github.com/everdev/mack"
	"github.com/spf13/cobra"
)

var (
	serverStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Starts corectld",
		RunE:  serverStartCommand,
	}
	shutdownCmd = &cobra.Command{
		Use:     "stop",
		Aliases: []string{"shutdown"},
		Short:   "Stops corectld",
		RunE:    shutdownCommand,
	}
	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Shows corectld status",
		RunE:  shared.PScommand,
	}
)

func shutdownCommand(cmd *cobra.Command, args []string) (err error) {
	if _, err = server.Daemon.Running(); err != nil {
		return
	}
	server.Query("server:stop", nil)
	return
}

func serverStartCommand(cmd *cobra.Command, args []string) (err error) {
	var srv interface{}

	if srv, err = server.Daemon.Running(); err == nil {
		return fmt.Errorf("corectld already started (with pid %v)",
			srv.(*release.Info).Pid)
	}

	if !session.Caller.Privileged {
		if err = mack.Tell("System Events",
			"do shell script \""+
				session.Executable()+" start --user "+session.Caller.Username+
				" > /dev/null 2>&1 & \" with administrator privileges",
			"delay 3"); err != nil {
			return
		}
		if srv, err = server.Daemon.Running(); err != nil {
			return err
		}
		if _, err = mack.AlertBox(mack.AlertOptions{
			Title: fmt.Sprintf("corectld (%v) just started with Pid %v .",
				srv.(*release.Info).Version, srv.(*release.Info).Pid),
			Message:  "\n\n(this window will self destruct after 15s)",
			Style:    "informational",
			Duration: 15,
			Buttons:  "OK"}); err != nil {
			return
		}
		fmt.Println("Started corectld:")
		srv.(*release.Info).PrettyPrint(true)
		return
	}
	server.Daemon = server.New()
	server.Daemon.Active = make(map[string]*server.VMInfo)
	return server.Start()
}

func init() {
	serverStartCmd.Flags().StringP("user", "u", "",
		"sets the user that will 'own' the corectld instance")
	serverStartCmd.Flags().BoolP("force", "f", false,
		"rebuilds config drive iso even if a suitable one is already present")
	rootCmd.AddCommand(shutdownCmd, statusCmd, serverStartCmd)
}
