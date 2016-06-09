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
	"github.com/TheNewNormal/corectl/release/cli"
	"github.com/helm/helm/log"
	"github.com/spf13/cobra"
)

var rootCmd = shared.RootCmdTmpl

// Main cli entry point
func Main() (err error) {
	return shared.STARTup(rootCmd)
}

func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) (err error) {
		cli := session.Caller.CmdLine
		cli.BindPFlags(cmd.Flags())
		if cli.GetBool("debug") {
			log.IsDebugging = true
		}
		if session.Caller.Privileged {
			return fmt.Errorf("too many privileges invoking %v, "+
				"please call it as a regular user", session.AppName())
		}
		return session.Caller.NormalizeOnDiskLayout()
	}
	shared.InitTmpl(rootCmd)
}
