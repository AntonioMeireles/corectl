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
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"sort"
	"strings"

	"github.com/deis/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/TheNewNormal/corectl/host/session"
	"github.com/TheNewNormal/corectl/server"
)

var (
	loadFCmd = &cobra.Command{
		Use:   "load path/to/yourProfile",
		Short: "Loads CoreOS instances defined in an instrumentation file.",
		Long: "Loads CoreOS instances defined in an instrumentation file " +
			"(either in TOML, JSON or YAML format).\n" + "VMs are always launched " +
			"by alphabetical order relative to their names.",
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) != 1 {
				return fmt.Errorf("Incorrect usage: " +
					"This command requires one argument (a file path)")
			}
			session.Caller.CmdLine.BindPFlags(cmd.Flags())
			return
		},
		RunE:    loadCommand,
		Example: `  corectl load profiles/demo.toml`,
	}
)

func loadCommand(cmd *cobra.Command, args []string) (err error) {
	var (
		vmDefs  = make(map[string]*viper.Viper)
		ordered []string
		f       []byte
		def     = args[0]
		setup   = viper.New()
	)

	if f, err = ioutil.ReadFile(def); err != nil {
		return
	}

	if strings.HasSuffix(def, ".toml") {
		setup.SetConfigType("toml")
	} else if strings.HasSuffix(def, ".json") {
		setup.SetConfigType("json")
	} else if strings.HasSuffix(def, ".yaml") ||
		strings.HasSuffix(def, ".yml") {
		setup.SetConfigType("yaml")
	} else {
		return fmt.Errorf("%s unable to guess format via suffix", def)
	}

	if err = setup.ReadConfig(bytes.NewBuffer(f)); err != nil {
		return
	}

	for name, def := range setup.AllSettings() {
		if reflect.ValueOf(def).Kind() == reflect.Map {
			lf := pflag.NewFlagSet(name, 0)
			runFlagsDefaults(lf)
			vmDefs[name] = viper.New()
			vmDefs[name].BindPFlags(lf)

			for x, xx := range setup.AllSettings() {
				if reflect.ValueOf(x).Kind() != reflect.Map {
					vmDefs[name].Set(x, xx)
				}
			}
			for x, xx := range def.(map[string]interface{}) {
				vmDefs[name].Set(x, xx)
			}
			vmDefs[name].Set("name", name)
		}
	}
	// (re)order alphabeticaly order to ensure cheap deterministic boot ordering
	for name := range vmDefs {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)
	for slot, name := range ordered {
		var (
			i  interface{}
			vm *server.VMInfo
		)

		fmt.Println("> booting", name, slot)
		if vm, err = vmBootstrap(vmDefs[name]); err != nil {
			return
		}
		if i, err = server.Query("vm:run", vm); err != nil {
			return
		}
		log.Info("'%v' started successfuly with address %v and PID %v",
			vm.Name, i.(*server.VMInfo).PublicIP, i.(*server.VMInfo).Pid)
		log.Info("'%v' boot logs can be found at '%v'", vm.Name, vm.Log())
	}
	return
}

func init() {
	rootCmd.AddCommand(loadFCmd)
}
