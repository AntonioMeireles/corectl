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
	"os"
	"text/tabwriter"

	"github.com/TheNewNormal/corectl/host/session"
	shared "github.com/TheNewNormal/corectl/release/cli"
	"github.com/TheNewNormal/corectl/server"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var (
	psCmd = &cobra.Command{
		Use:     "ps",
		Short:   "Lists running CoreOS instances",
		PreRunE: shared.DefaultPreRunE,
		RunE:    shared.PScommand,
	}
	queryCmd = &cobra.Command{
		Use:     "query [VMids]",
		Aliases: []string{"q"},
		Short:   "Display information about the running CoreOS instances",
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			session.Caller.CmdLine.BindPFlags(cmd.Flags())
			if (session.Caller.CmdLine.GetBool("ip") ||
				session.Caller.CmdLine.GetBool("tty") ||
				session.Caller.CmdLine.GetBool("log")) && len(args) != 1 {
				err = fmt.Errorf("Incorrect Usage: only one argument expected " +
					"(a VM's name or UUID)")
			}
			return err
		},
		RunE: queryCommand,
	}
)

func queryCommand(cmd *cobra.Command, args []string) (err error) {
	var (
		pp       []byte
		i        interface{}
		selected map[string]*server.VMInfo
		vm       *server.VMInfo
		tabP     = func(selected map[string]*server.VMInfo) {
			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 5, 0, 1, ' ', 0)
			fmt.Fprintf(w, "name\tchannel/version\tip\tcpu(s)\tram\tuuid\tpid"+
				"\tuptime\tvols\n")
			for _, vm := range selected {
				fmt.Fprintf(w, "%v\t%v/%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
					vm.Name, vm.Channel, vm.Version, vm.PublicIP, vm.Cpus,
					vm.Memory, vm.UUID, vm.Pid, humanize.Time(vm.CreationTime),
					len(vm.Storage.HardDrives))
			}
			w.Flush()
		}
	)

	cli := session.Caller.CmdLine
	if _, err = server.Daemon.Running(); err != nil {
		return session.ErrServerUnreachable
	}

	if i, err = server.Query("vm:list", nil); err != nil {
		return
	}
	running := i.(map[string]*server.VMInfo)

	if len(args) == 1 {
		if vm, err = vmInfo(args[0]); err != nil {
			return
		}
		if cli.GetBool("ip") {
			fmt.Println(vm.PublicIP)
			return
		} else if cli.GetBool("tty") {
			fmt.Println(vm.TTY())
			return
		} else if cli.GetBool("log") {
			fmt.Println(vm.Log())
			return
		}
	}

	if len(args) == 0 {
		selected = running
	} else {
		selected = make(map[string]*server.VMInfo)
		for _, target := range args {
			if vm, err = vmInfo(target); err != nil {
				return
			}
			selected[vm.UUID] = vm
		}
	}

	if cli.GetBool("json") {
		if pp, err = json.MarshalIndent(selected, "", "    "); err == nil {
			fmt.Println(string(pp))
		}
	} else if cli.GetBool("all") {
		tabP(selected)
	} else {
		for _, vm := range selected {
			fmt.Println(vm.Name)
		}
	}
	return
}

func init() {
	psCmd.Flags().BoolP("json", "j", false,
		"outputs in JSON for easy 3rd party integration")
	rootCmd.AddCommand(psCmd)

	queryCmd.Flags().BoolP("json", "j", false,
		"outputs in JSON for easy 3rd party integration")
	queryCmd.Flags().BoolP("all", "a", false,
		"display a table with extended information about running "+
			"CoreOS instances")
	queryCmd.Flags().BoolP("ip", "i", false,
		"displays given instance IP address")
	queryCmd.Flags().BoolP("tty", "t", false,
		"displays given instance tty's location")
	queryCmd.Flags().BoolP("log", "l", false,
		"displays given instance boot logs location")
	rootCmd.AddCommand(queryCmd)
}
