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
	"strings"

	"github.com/TheNewNormal/corectl/host/session"
	"github.com/TheNewNormal/corectl/server"
	"github.com/TheNewNormal/corectl/target/coreos"
	"github.com/helm/helm/log"
	"github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	runCmd = &cobra.Command{
		Use:     "run",
		Aliases: []string{"start"},
		Short:   "Boots a new CoreOS instance",
		RunE:    runCommand,
	}
)

func runCommand(cmd *cobra.Command, args []string) (err error) {
	var (
		i   interface{}
		vm  *server.VMInfo
		cli = session.Caller.CmdLine
	)

	if _, err = server.Daemon.Running(); err != nil {
		return session.ErrServerUnreachable
	}
	if vm, err = vmBootstrap(cli); err != nil {
		return
	}
	if i, err = server.Query("vm:run", vm); err != nil {
		return
	}
	log.Info("'%v' started successfuly with address %v and PID %v",
		vm.Name, i.(*server.VMInfo).PublicIP, i.(*server.VMInfo).Pid)
	log.Info("'%v' boot logs can be found at '%v'", vm.Name, vm.Log())
	log.Info("'%v' console can be found at '%v'", vm.Name, vm.TTY())
	return
}

func vmBootstrap(args *viper.Viper) (vm *server.VMInfo, err error) {
	pSlice := func(plain []string) []string {
		// getting around https://github.com/spf13/viper/issues/112
		var sliced []string
		for _, x := range plain {
			strip := strings.Replace(
				strings.Replace(x, "]", "", -1), "[", "", -1)
			for _, y := range strings.Split(strip, ",") {
				sliced = append(sliced, y)
			}
		}
		return sliced
	}
	vm = new(server.VMInfo)

	vm.OfflineMode = args.GetBool("offline")
	vm.Cpus = args.GetInt("cpus")
	vm.AddToHypervisor = args.GetString("extra")
	vm.AddToKernel = args.GetString("boot")
	vm.SSHkey = args.GetString("sshkey")
	vm.SharedHomedir = args.GetBool("shared-homedir")
	vm.Root = -1
	vm.Pid = -1

	vm.Name = args.GetString("name")
	vm.UUID = args.GetString("uuid")

	if vm.UUID == "random" {
		vm.UUID = uuid.NewV4().String()
	} else if _, err = uuid.FromString(vm.UUID); err != nil {
		log.Warn("%s not a valid UUID as it doesn't follow RFC "+
			"4122. %s\n", vm.UUID, "Using a randomly generated one")
		vm.UUID = uuid.NewV4().String()
	}

	if vm.Name == "" {
		vm.Name = vm.UUID
	}

	var i interface{}
	if i, err = server.Query("vm:uuid2mac",
		[]string{vm.UUID, args.GetString("uuid")}); err != nil {
		return
	}
	vm.MacAddress, vm.UUID = i.([]string)[0], i.([]string)[1]

	vm.Memory = args.GetInt("memory")
	if vm.Memory < 1024 {
		log.Warn("'%v' not a reasonable memory value. %s\n", vm.Memory,
			"Using '1024', the default")
		vm.Memory = 1024
	} else if vm.Memory > 8192 {
		log.Warn("'%v' not a reasonable memory value, as presently "+
			"we only support VMs with up to 8GB of RAM. setting "+
			"it to '8192'", vm.Memory)
		vm.Memory = 8192
	}

	vm.Channel = coreos.Channel(args.GetString("channel"))

	vm.Version = coreos.Version(args.GetString("version"))
	vm.Version, err =
		server.PullImage(vm.Channel, vm.Version, false, vm.OfflineMode)
	if err != nil {
		return
	}

	vm.ValidateCDROM(session.Caller.ConfigISO())

	if err = vm.ValidateVolumes([]string{args.GetString("root")},
		true); err != nil {
		return
	}
	if err = vm.ValidateVolumes(pSlice(args.GetStringSlice("volume")),
		false); err != nil {
		return
	}

	vm.Ethernet =
		append(vm.Ethernet, server.NetworkInterface{Type: server.Raw})

	err = vm.ValidateCloudConfig(args.GetString("cloud_config"))
	if err != nil {
		return
	}

	if err = vm.SSHkeyGen(); err != nil {
		err = fmt.Errorf("Aborting: unable to generate internal SSH "+
			"key pair (!) (%v)", err)
	}

	return
}

func runFlagsDefaults(setFlag *pflag.FlagSet) {
	setFlag.StringP("channel", "c", "alpha", "CoreOS channel stream")
	setFlag.StringP("version", "v", "latest", "CoreOS version")
	setFlag.StringP("uuid", "u", "random", "VM's UUID")
	setFlag.IntP("memory", "m", 1024,
		"VM's RAM, in MB, per instance (1024 < memory < 8192)")
	setFlag.IntP("cpus", "N", 1, "VM number of virtual CPUs")
	setFlag.StringP("cloud_config", "L", "",
		"cloud-config file location (either an URL or a local path)")
	setFlag.StringP("sshkey", "k", "", "VM's default ssh key")
	setFlag.StringP("root", "r", "", "append a (persistent) root volume to VM")
	setFlag.StringSliceP("volume", "p", nil, "append disk volumes to VM")
	setFlag.BoolP("offline", "o", false,
		"doesn't go online to check for newer images than the "+
			"locally available ones unless there is none available.")
	setFlag.StringP("name", "n", "",
		"names the VM (default is VM's UUID)")
	setFlag.BoolP("shared-homedir", "H", false,
		"mounts (via NFS) host's homedir inside VM")
	setFlag.StringP("extra", "x", "", "additional arguments to the hypervisor")
	setFlag.StringP("boot", "b", "", "additional arguments to the kernel boot")
	// available but hidden...
	setFlag.StringP("tap", "t", "", "append tap interface to VM")
	setFlag.MarkHidden("tap")
}

func init() {
	runFlagsDefaults(runCmd.Flags())
	rootCmd.AddCommand(runCmd)
}
