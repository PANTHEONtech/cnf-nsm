/*
 * Copyright (c) 2020 PANTHEON.tech s.r.o. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"github.com/contiv/vpp/plugins/crd"
	"go.ligato.io/cn-infra/v2/agent"
	"go.ligato.io/cn-infra/v2/datasync/resync"
	"go.ligato.io/cn-infra/v2/db/keyval/etcd"
	"go.ligato.io/cn-infra/v2/health/probe"
	"go.ligato.io/cn-infra/v2/logging/logrus"

	// load all VPP-agent models for CNFConfiguration CRD handler to use
	_ "go.ligato.io/vpp-agent/v3/proto/ligato/linux"
	_ "go.ligato.io/vpp-agent/v3/proto/ligato/linux/iptables"
	_ "go.ligato.io/vpp-agent/v3/proto/ligato/vpp"
	_ "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/srv6"

	"go.cdnf.io/cnf-nsm/plugins/crdplugin"
)

// CNFCRD is a Kubernetes CRD controller providing additional features for the sake of Pantheon CNFs.
type CNFCRD struct {
	HealthProbe *probe.Plugin
	CRD         *crdplugin.Plugin
}

func (c *CNFCRD) String() string {
	return "CNF-CRD"
}

// Init is called at startup phase. Method added in order to implement Plugin interface.
func (c *CNFCRD) Init() error {
	return nil
}

// AfterInit triggers the first resync.
func (c *CNFCRD) AfterInit() error {
	resync.DefaultPlugin.DoResync()
	return nil
}

// Close is called at cleanup phase. Method added in order to implement Plugin interface.
func (c *CNFCRD) Close() error {
	return nil
}

func main() {
	// disable status check for etcd
	etcd.DefaultPlugin.StatusCheck = nil

	crd.DefaultPlugin.Etcd = &etcd.DefaultPlugin
	probe.DefaultPlugin.NonFatalPlugins = []string{"etcd"}

	CNFCRD := &CNFCRD{
		HealthProbe: &probe.DefaultPlugin,
		CRD:         &crdplugin.DefaultPlugin,
	}

	a := agent.NewAgent(agent.AllPlugins(CNFCRD))
	if err := a.Run(); err != nil {
		logrus.DefaultLogger().Fatal(err)
	}
}
