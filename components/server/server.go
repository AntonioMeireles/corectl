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
	"os/signal"
	"sync"
	"syscall"

	"github.com/TheNewNormal/corectl/components/host/session"
	"github.com/TheNewNormal/corectl/release"
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
