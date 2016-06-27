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

package server

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/TheNewNormal/corectl/host/darwin/misc/uuid2ip"
	"github.com/TheNewNormal/corectl/host/session"
	"github.com/TheNewNormal/corectl/release"
	"github.com/blang/semver"
	"github.com/deis/pkg/log"
	"github.com/satori/go.uuid"
	"github.com/valyala/gorpc"
)

func RPCservices() {
	gorpc.RegisterType(&release.Info{})
	gorpc.RegisterType(&VMInfo{})

	session.Caller.Services.AddFunc("echo", echo)
	session.Caller.Services.AddFunc("images:list", availableImages)
	session.Caller.Services.AddFunc("images:remove", removeImage)
	session.Caller.Services.AddFunc("vm:list", activeVMs)
	session.Caller.Services.AddFunc("vm:run", run)
	session.Caller.Services.AddFunc("vm:stop", stopVMs)
	session.Caller.Services.AddFunc("vm:uuid2mac", uuid2mac)
	session.Caller.Services.AddFunc("server:stop", shutdown)
}

func (cfg *Config) Running() (interface{}, error) {
	return session.Caller.RPCdispatcher.CallTimeout("echo",
		nil, 100*time.Millisecond)
}

func Query(funcName string, request interface{}) (interface{}, error) {
	return session.Caller.RPCdispatcher.CallTimeout(funcName, request,
		ServerTimeout)
}

func shutdown() (err error) {
	log.Debug("server:stop")
	log.Info("Sky must be falling. Shutting down...")

	Daemon.RPCserver.Stop()
	return
}
func run(vm *VMInfo) (booted *VMInfo, err error) {
	log.Debug("vm:run")

	var bootArgs []string

	vm.publicIPCh = make(chan string, 1)
	vm.errCh = make(chan error, 1)
	vm.done = make(chan struct{})

	if err = vm.register(); err != nil {
		return
	}

	if bootArgs, err = vm.assembleBootPayload(); err != nil {
		return
	}
	if err = vm.MkRunDir(); err != nil {
		return
	}
	vm.CreationTime = time.Now()

	payload := append(strings.Split(bootArgs[0], " "),
		"-f", fmt.Sprintf("%s%v", bootArgs[1], bootArgs[2]))
	vm.exec = exec.Command(filepath.Join(session.ExecutableFolder(),
		"corectld.runner"), payload...)

	go func() {
		timeout := time.After(ServerTimeout)
		select {
		case <-timeout:
			vm.Pid = vm.exec.Process.Pid
			vm.halt()
			vm.errCh <- fmt.Errorf("Unable to grab VM's IP after " +
				"30s (!)... Aborted")
		case ip := <-vm.publicIPCh:
			close(vm.publicIPCh)
			close(vm.done)
			vm.Pid, vm.PublicIP = vm.exec.Process.Pid, ip
			log.Info("started '%s' in background with IP %v and "+
				"PID %v\n", vm.Name, vm.PublicIP, vm.exec.Process.Pid)
		}
	}()

	go func() {
		Daemon.Jobs.Add(1)
		defer Daemon.Jobs.Done()
		if err := vm.exec.Start(); err != nil {
			vm.errCh <- err
		}
		vm.exec.Wait()
		vm.deregister()
		os.Remove(vm.TTY())
		// give it time to flush logs
		time.Sleep(3 * time.Second)
	}()

	select {
	case <-vm.done:
		if len(vm.PublicIP) == 0 {
			err = fmt.Errorf("VM terminated abnormally too early")
		}
		return vm, err
	case err = <-vm.errCh:
		return
	}

}

func uuid2mac(input []string) (output []string, err error) {
	log.Debug("vm:uuid2mac")
	var (
		MAC            string
		UUID, original = input[0], input[1]
	)

	// handles UUIDs
	if _, ok := Daemon.Active[UUID]; ok {
		err = fmt.Errorf("Aborted: Another VM is "+
			"already running with the exact same UUID (%s)",
			UUID)
	} else {
		for {
			// we keep the loop just in case as the check
			// above is supposed to avoid most failures...
			// XXX
			if MAC, err =
				uuid2ip.GuestMACfromUUID(UUID); err == nil {
				// if ip, err = uuid2ip.GuestIPfromMAC(MAC); err == nil {
				// 	log.Info("GUEST IP will be %v", ip)
				break
				// }
			}
			fmt.Println("=>", original, err)
			if original != "random" {
				log.Warn("unable to guess the MAC Address from the provided "+
					"UUID (%s). Using a randomly generated one\n", original)
			}
			UUID = uuid.NewV4().String()
		}
	}
	output = []string{MAC, UUID}
	return
}

func stopVMs(targets []string) error {
	log.Debug("vm:stop")

	var toHalt []string

	if len(targets) == 0 {
		for _, x := range Daemon.Active {
			toHalt = append(toHalt, x.UUID)
		}
	} else {
		for _, t := range targets {
			for _, v := range Daemon.Active {
				if v.Name == t || v.UUID == t {
					toHalt = append(toHalt, v.UUID)
				}
			}
		}
	}
	for _, v := range toHalt {
		Daemon.Active[v].halt()
	}
	return nil
}

func activeVMs() (running map[string]*VMInfo) {
	log.Debug("vm:list")
	return Daemon.Active
}

func availableImages() (map[string]semver.Versions, error) {
	log.Debug("images:list")
	Daemon.Lock()
	defer Daemon.Unlock()
	return localImages()
}

func removeImage(args []string) (available map[string]semver.Versions, err error) {
	log.Debug("images:remove")
	Daemon.Lock()

	channel, version := args[0], args[1]
	var x int
	var y semver.Version
	for x, y = range Daemon.Media[channel] {
		if version == y.String() {
			break
		}
	}
	log.Debug("removing %v/%v", channel, version)
	Daemon.Media[channel] = append(Daemon.Media[channel][:x],
		Daemon.Media[channel][x+1:]...)
	Daemon.Unlock()
	log.Debug("%s/%s was made unavailable", channel, version)
	if err = os.RemoveAll(path.Join(session.Caller.ImageStore(),
		channel, y.String())); err != nil {
		log.Err(err.Error())
		return
	}
	return localImages()
}

func echo() *release.Info {
	log.Debug("ping")
	return Daemon.Meta
}
