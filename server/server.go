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
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/TheNewNormal/corectl/host/session"
	"github.com/TheNewNormal/corectl/release"
	"github.com/TheNewNormal/corectl/target/coreos"
	"github.com/blang/semver"
	"github.com/helm/helm/log"
	"github.com/valyala/gorpc"
)

type (
	// MediaAssets ...
	MediaAssets map[string]semver.Versions

	// Config ...
	Config struct {
		sync.Mutex
		Meta      *release.Info
		Media     MediaAssets
		Active    map[string]*VMInfo
		RPCserver *gorpc.Server
		Jobs      sync.WaitGroup
	}
)

var Daemon *Config

// New ...
func New() *Config {
	return &Config{
		Meta:      session.Caller.Meta,
		Media:     nil,
		Active:    nil,
		RPCserver: nil,
		Jobs:      sync.WaitGroup{},
	}
}

// Start server...
func Start() (err error) {
	// var  closeVPNhooks func()
	if !session.Caller.Privileged {
		return fmt.Errorf("not enough previleges to start server. " +
			"please use 'sudo'")
	}

	log.Info("checking nfs host settings")
	if err = nfsSetup(); err != nil {
		return
	}
	// log.Info("checking for VPN setups")
	// if closeVPNhooks, err = HandleVPNtunnels(); err != nil {
	// 	return
	// }
	// defer closeVPNhooks()

	log.Info("registering locally available images")
	if Daemon.Media, err = localImages(); err != nil {
		return
	}
	hades := make(chan os.Signal, 1)
	signal.Notify(hades,
		os.Interrupt,
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-hades
		log.Info("Got '%v' signal, stopping server...", s)
		signal.Stop(hades)
		Daemon.RPCserver.Stop()
	}()

	log.Info("server starting...")

	Daemon.RPCserver = gorpc.NewTCPServer("127.0.0.1:2511",
		session.Caller.Services.NewHandlerFunc())
	if err = Daemon.RPCserver.Serve(); err != nil {
		log.Err("Cannot start RPC server [%s]", err)
		return
	}
	for _, r := range Daemon.Active {
		r.halt()
	}
	Daemon.Jobs.Wait()

	log.Info("gone!")
	return
}

func ConfigDrive() (err error) {
	var tmpDir, user_data string

	if tmpDir, err = ioutil.TempDir(session.Caller.TmpDir(),
		"coreos"); err != nil {
		return
	}

	defer func() {
		os.RemoveAll(tmpDir)
	}()

	user_data = path.Join(tmpDir, "openstack/latest/user_data")
	if err = os.MkdirAll(filepath.Dir(user_data), 0644); err != nil {
		return
	}

	h, _ := os.Create(user_data)
	w := bufio.NewWriter(h)
	defer h.Close()

	fmt.Fprintf(w, coreos.CoreOEMsetupBootstrap)
	w.Flush()

	cmd := exec.Command("hdiutil", "makehybrid", "-iso", "-joliet", "-ov",
		"-default-volume-name", "config-2",
		"-o", session.Caller.ConfigISO(), tmpDir)

	return cmd.Run()
}
